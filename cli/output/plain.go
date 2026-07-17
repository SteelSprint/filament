package output

import (
	"fmt"
	"strconv"
	"strings"

	"drift/core"
	"drift/internal/diff"
	"drift/orchestrator"
)

// PlainPresenter is the byte-identical continuation of pre-output-layer output.
// It never emits ANSI sequences and never produces JSON. Methods are pure
// migrations of the format* functions that lived in cli/cli.go before Landing 1
// of the output-layer refactor (see PLAN.md). All file I/O is resolved by the
// dispatch before constructing a Result; PlainPresenter only formats.
type PlainPresenter struct{}

// markerSyntax is the user-facing shorthand for marker comments.
var markerSyntax = "D" + "! id=<markerid>"

// D! id=cfmt range-start
func (p PlainPresenter) Todo(r TodoResult) string {
	state := r.State
	if len(state.Specs) == 0 && len(state.Markers) == 0 {
		return "Nothing to check: no specs or markers registered.\nCreate spec files (*.drift.xml) and place " + markerSyntax + " markers in your code,\nthen run `drift link <marker> <module.spec>` to connect them."
	}

	var sb strings.Builder

	if len(state.Todos) == 0 {
		sb.WriteString(fmt.Sprintf("No changes detected. %d specs, %d markers, %d links in sync.", len(state.Specs), len(state.Markers), len(state.Links)))
	} else {
		changedMarkers := make(map[string]bool)
		changedSpecs := make(map[string]bool)
		for _, todo := range state.Todos {
			if todo.MarkerChanged {
				changedMarkers[todo.MarkerID] = true
			}
			if todo.SpecChanged {
				changedSpecs[todo.SpecID] = true
			}
		}

		if n := len(changedMarkers); n > 0 {
			if n == 1 {
				sb.WriteString("1 marker has unchecked changes.\n")
			} else {
				sb.WriteString(fmt.Sprintf("%d markers have unchecked changes.\n", n))
			}
		}
		if n := len(changedSpecs); n > 0 {
			if n == 1 {
				sb.WriteString("1 spec item has unchecked changes.\n")
			} else {
				sb.WriteString(fmt.Sprintf("%d spec items have unchecked changes.\n", n))
			}
		}

		sb.WriteString("\n")

		for i, todo := range state.Todos {
			var driftDescription string
			switch {
			case todo.SpecDeleted:
				driftDescription = "The spec term has been deleted from disk. If this was intentional, run the reset command below to acknowledge the removal."
			case todo.MarkerDeleted:
				driftDescription = "The marker has been deleted from disk. If this was intentional, run the reset command below to acknowledge the removal."
			case todo.MarkerChanged && todo.SpecChanged:
				driftDescription = "Both the marker and the spec term have changed. Please check whether the changed code still complies with the new version of the spec term and make any modifications necessary on either side."
			case todo.MarkerChanged:
				driftDescription = "The marker has changed but not the spec term. Please check whether the changed code still complies with the spec term and make any modifications necessary."
			default:
				driftDescription = "The spec term has changed but not the marker. Please check whether the new version of the spec term is still reflected in the code and make any modifications necessary."
			}

			markerLocation := todo.MarkerFilepath + ":" + strconv.Itoa(todo.MarkerLineNumber)
			specLocation := todo.SpecFilepath + ":" + strconv.Itoa(todo.SpecLineNumber)

			sb.WriteString(fmt.Sprintf("%d. [TODO] Edge between marker %q in %q and spec term %q in %q. %s Once you are satisfied, run `drift reset %s %s` to mark this todo item as complete.\n",
				i+1,
				todo.MarkerID,
				markerLocation,
				todo.SpecID,
				specLocation,
				driftDescription,
				todo.MarkerID,
				todo.SpecID,
			))
			sb.WriteString(fmt.Sprintf("  → Run 'drift diff %s %s' to see what changed.\n", todo.MarkerID, todo.SpecID))
		}
	}

	if warning := unlinkedMarkerWarning(state); warning != "" {
		sb.WriteString("\n")
		sb.WriteString(warning)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// unlinkedMarkerWarning returns the one-line warning summary for non-deleted
// markers that have no links, or "" when there are none.
func unlinkedMarkerWarning(state core.EvaluatedState) string {
	linkedMarkers := make(map[string]bool, len(state.Links))
	for _, link := range state.Links {
		linkedMarkers[link.MarkerID] = true
	}
	unlinked := 0
	for _, m := range state.Markers {
		if m.Deleted {
			continue
		}
		if !linkedMarkers[m.ID] {
			unlinked++
		}
	}
	if unlinked == 0 {
		return ""
	}
	if unlinked == 1 {
		return "1 unlinked marker found — run `drift list` to review."
	}
	return fmt.Sprintf("%d unlinked markers found — run `drift list` to review.", unlinked)
}

// D! id=cfmt range-end

// D! id=ofmtl range-start
func (p PlainPresenter) List(r ListResult) string {
	state := r.State
	if len(state.Specs) == 0 && len(state.Markers) == 0 {
		return "No specs or markers registered.\nRun `drift init` to get started, then create spec files (*.drift.xml) and place " + markerSyntax + " markers in your code."
	}

	driftedEdges := make(map[string]bool)
	for _, todo := range state.Todos {
		driftedEdges[todo.MarkerID+"\x00"+todo.SpecID] = true
	}

	linkedSpecs := make(map[string]bool)
	linkedMarkers := make(map[string]bool)
	for _, link := range state.Links {
		linkedSpecs[link.SpecID] = true
		linkedMarkers[link.MarkerID] = true
	}

	var sb strings.Builder

	sortedSpecs := make([]core.Spec, len(state.Specs))
	copy(sortedSpecs, state.Specs)
	sortSpecsByID(sortedSpecs)

	sb.WriteString(fmt.Sprintf("Specs (%d):\n", len(sortedSpecs)))
	for _, spec := range sortedSpecs {
		linkFlag := ""
		if spec.Deleted {
			linkFlag = " [deleted]"
		} else if !linkedSpecs[spec.ID] {
			linkFlag = " [unlinked]"
		}
		sb.WriteString(fmt.Sprintf("  %-30s %s%s\n", spec.ID, spec.Filepath, linkFlag))
		if r.Verbose && !spec.Deleted {
			if content, ok := r.SpecContents[spec.ID]; ok && len(content) > 0 {
				preview := content
				if len(preview) > 80 {
					preview = preview[:80] + "..."
				}
				sb.WriteString(fmt.Sprintf("    %s\n", preview))
			}
		}
	}

	sortedMarkers := make([]core.Marker, len(state.Markers))
	copy(sortedMarkers, state.Markers)
	sortMarkersByID(sortedMarkers)

	sb.WriteString(fmt.Sprintf("\nMarkers (%d):\n", len(sortedMarkers)))
	for _, marker := range sortedMarkers {
		linkFlag := ""
		if marker.Deleted {
			linkFlag = " [deleted]"
		} else if !linkedMarkers[marker.ID] {
			linkFlag = " [unlinked]"
		}
		sb.WriteString(fmt.Sprintf("  %-30s %s:%d-%d%s\n", marker.ID, marker.Filepath, marker.LineNumber, marker.EndLineNumber, linkFlag))
		if r.Verbose && !marker.Deleted {
			if content, ok := r.MarkerContents[marker.ID]; ok && len(content) > 0 {
				firstLine := strings.Split(content, "\n")[0]
				if len(firstLine) > 80 {
					firstLine = firstLine[:80] + "..."
				}
				if firstLine != "" {
					sb.WriteString(fmt.Sprintf("    %s\n", firstLine))
				}
			}
		}
	}

	if len(state.Links) > 0 {
		sb.WriteString(fmt.Sprintf("\nLinks (%d):\n", len(state.Links)))
		for _, link := range state.Links {
			status := "[synced]"
			if driftedEdges[link.MarkerID+"\x00"+link.SpecID] {
				status = "[DRIFTED]"
			}
			sb.WriteString(fmt.Sprintf("  %-15s → %-30s %s\n", link.MarkerID, link.SpecID, status))
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func sortSpecsByID(specs []core.Spec) {
	for i := 1; i < len(specs); i++ {
		for j := i; j > 0 && specs[j-1].ID > specs[j].ID; j-- {
			specs[j], specs[j-1] = specs[j-1], specs[j]
		}
	}
}

func sortMarkersByID(markers []core.Marker) {
	for i := 1; i < len(markers); i++ {
		for j := i; j > 0 && markers[j-1].ID > markers[j].ID; j-- {
			markers[j], markers[j-1] = markers[j-1], markers[j]
		}
	}
}

// D! id=ofmtl range-end

func (p PlainPresenter) Show(r ShowResult) string {
	if r.IsSpec {
		if r.Spec == nil {
			return fmt.Sprintf("spec %q not found", r.ID)
		}
		return p.showSpec(r)
	}
	if r.Marker == nil {
		return fmt.Sprintf("marker %q not found", r.ID)
	}
	return p.showMarker(r)
}

func (p PlainPresenter) showSpec(r ShowResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== Spec: %s ===\n", r.ID))
	sb.WriteString(fmt.Sprintf("File: %s\n", r.Spec.Filepath))
	sb.WriteString(fmt.Sprintf("Hash: %s\n\n", r.Spec.Hash))
	sb.WriteString(r.Content)
	sb.WriteString("\n")

	for _, m := range r.LinkedMarkers {
		sb.WriteString(fmt.Sprintf("\n=== Marker: %s ===\n", m.Marker.ID))
		sb.WriteString(fmt.Sprintf("File: %s\n", m.Marker.Filepath))
		sb.WriteString(fmt.Sprintf("Lines: %d-%d\n", m.Marker.LineNumber, m.Marker.EndLineNumber))
		sb.WriteString(fmt.Sprintf("Hash: %s\n\n", m.Marker.Hash))
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func (p PlainPresenter) showMarker(r ShowResult) string {
	var sb strings.Builder

	for _, s := range r.LinkedSpecs {
		sb.WriteString(fmt.Sprintf("=== Spec: %s ===\n", s.Spec.ID))
		sb.WriteString(fmt.Sprintf("File: %s\n", s.Spec.Filepath))
		sb.WriteString(fmt.Sprintf("Hash: %s\n\n", s.Spec.Hash))
		sb.WriteString(s.Content)
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("=== Marker: %s ===\n", r.ID))
	sb.WriteString(fmt.Sprintf("File: %s\n", r.Marker.Filepath))
	sb.WriteString(fmt.Sprintf("Lines: %d-%d\n", r.Marker.LineNumber, r.Marker.EndLineNumber))
	sb.WriteString(fmt.Sprintf("Hash: %s\n\n", r.Marker.Hash))
	sb.WriteString(r.Content)

	return strings.TrimRight(sb.String(), "\n")
}

// D! id=cdifffmt range-start
func (p PlainPresenter) DiffEdge(r DiffEdgeResult) string {
	var sb strings.Builder
	sb.WriteString(p.formatDiffSide("Spec", r.Result.Spec))
	sb.WriteString("\n---\n")
	sb.WriteString(p.formatDiffSide("Marker", r.Result.Marker))
	return strings.TrimRight(sb.String(), "\n")
}

func (p PlainPresenter) formatDiffSide(label string, side orchestrator.DiffSide) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s", label, side.ID))
	if side.Filepath != "" {
		sb.WriteString(fmt.Sprintf(" (%s", side.Filepath))
		if side.Lines != "" {
			sb.WriteString(":" + side.Lines)
		}
		sb.WriteString(")")
	}
	sb.WriteString("\n")

	if side.Deleted {
		sb.WriteString("Status: deleted from disk\n")
	} else if !side.HasBaseline {
		sb.WriteString(fmt.Sprintf("Status: no baseline snapshot (hash %s)\n", side.BaselineHash))
	} else if side.BaselineHash == side.CurrentHash && side.CurrentHash != "" {
		sb.WriteString("Status: in sync\n")
	} else {
		sb.WriteString(fmt.Sprintf("Baseline: %s   Current: %s\n", side.BaselineHash, side.CurrentHash))
	}

	if !side.HasBaseline {
		return strings.TrimRight(sb.String(), "\n")
	}
	if side.Baseline == side.Current {
		return strings.TrimRight(sb.String(), "\n")
	}

	sb.WriteString("\n--- baseline\n+++ current\n")
	patch := diff.UnifiedDiff(side.Baseline, side.Current)
	if patch != "" {
		sb.WriteString(patch)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (p PlainPresenter) DiffExpanded(r DiffExpandedResult) string {
	var sb strings.Builder
	for i, edge := range r.Edges {
		if i > 0 {
			sb.WriteString("\n\n===\n\n")
		}
		out := p.DiffEdge(DiffEdgeResult{Result: edge})
		sb.WriteString(out)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// D! id=cdall range-start
// formatDiffAll shows the full unified diff for every drifted edge (every entry
// in state.Todos). Synced edges are NOT shown (use `drift list` for the full
// mapping). Returns "No drift detected." with exit code 0 when there are no
// todos. This is the review-friendly counterpart to the deliberately absent
// bulk reset.
func (p PlainPresenter) DiffAll(r DiffAllResult) string {
	if len(r.Edges) == 0 {
		return "No drift detected."
	}

	var sb strings.Builder
	for i, edge := range r.Edges {
		if i > 0 {
			sb.WriteString("\n\n===\n\n")
		}
		out := p.DiffEdge(DiffEdgeResult{Result: edge})
		sb.WriteString(out)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// D! id=cdall range-end
// D! id=cdifffmt range-end

func (p PlainPresenter) Ok(r OkResult) string {
	return r.Message
}

func (p PlainPresenter) Error(r ErrorResult) string {
	if r.Hint != "" {
		return r.Message + "\n" + r.Hint
	}
	return r.Message
}

func (p PlainPresenter) Text(r TextResult) string {
	return r.Text
}

func (p PlainPresenter) Version(r VersionResult) string {
	return "drift version " + r.Version
}
