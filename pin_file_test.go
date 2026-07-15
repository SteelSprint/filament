package driftpin

import (
	"os"
	"path/filepath"
	"testing"
)

func assertPinStateEquals(t *testing.T, got, want PinState) {
	t.Helper()
	if len(got.Specs) != len(want.Specs) {
		t.Fatalf("specs length = %d, want %d (got=%v want=%v)", len(got.Specs), len(want.Specs), got.Specs, want.Specs)
	}
	for i := range got.Specs {
		if got.Specs[i] != want.Specs[i] {
			t.Fatalf("spec[%d] = %+v, want %+v", i, got.Specs[i], want.Specs[i])
		}
	}
	if len(got.Markers) != len(want.Markers) {
		t.Fatalf("markers length = %d, want %d (got=%v want=%v)", len(got.Markers), len(want.Markers), got.Markers, want.Markers)
	}
	for i := range got.Markers {
		if got.Markers[i] != want.Markers[i] {
			t.Fatalf("marker[%d] = %+v, want %+v", i, got.Markers[i], want.Markers[i])
		}
	}
	if len(got.Links) != len(want.Links) {
		t.Fatalf("links length = %d, want %d (got=%v want=%v)", len(got.Links), len(want.Links), got.Links, want.Links)
	}
	for i := range got.Links {
		if got.Links[i] != want.Links[i] {
			t.Fatalf("link[%d] = %+v, want %+v", i, got.Links[i], want.Links[i])
		}
	}
	if len(got.ResolutionState) != len(want.ResolutionState) {
		t.Fatalf("resolutions length = %d, want %d (got=%v want=%v)", len(got.ResolutionState), len(want.ResolutionState), got.ResolutionState, want.ResolutionState)
	}
	for i := range got.ResolutionState {
		if got.ResolutionState[i] != want.ResolutionState[i] {
			t.Fatalf("resolution[%d] = %+v, want %+v", i, got.ResolutionState[i], want.ResolutionState[i])
		}
	}
}

func TestFilePinStoreRoundTrip(t *testing.T) {
	shapes := []struct {
		name  string
		state PinState
	}{
		{"empty", PinState{}},
		{"one_spec", PinState{
			Specs: []Spec{newSpec("s1", "h1")},
		}},
		{"one_marker", PinState{
			Markers: []Marker{newMarker("m1", "h1")},
		}},
		{"one_spec_one_marker_no_link", PinState{
			Specs:   []Spec{newSpec("s1", "h1")},
			Markers: []Marker{newMarker("m1", "h1")},
		}},
		{"one_spec_one_marker_one_link", PinState{
			Specs:   []Spec{newSpec("s1", "h1")},
			Markers: []Marker{newMarker("m1", "h1")},
			Links:   []Link{newLink("s1", "m1")},
		}},
		{"one_resolution", PinState{
			Specs:           []Spec{newSpec("s1", "h1")},
			Markers:         []Marker{newMarker("m1", "h1")},
			Links:           []Link{newLink("s1", "m1")},
			ResolutionState: []ResolutionState{newResolutionState("s1", "m1", "ch1", "ch2")},
		}},
		{"many_specs", PinState{
			Specs: []Spec{newSpec("s1", "h1"), newSpec("s2", "h2"), newSpec("s3", "h3")},
		}},
		{"many_markers", PinState{
			Markers: []Marker{newMarker("m1", "h1"), newMarker("m2", "h2"), newMarker("m3", "h3")},
		}},
		{"many_links_2x2", PinState{
			Specs:   []Spec{newSpec("s1", "h1"), newSpec("s2", "h2")},
			Markers: []Marker{newMarker("m1", "h1"), newMarker("m2", "h2")},
			Links:   []Link{newLink("s1", "m1"), newLink("s1", "m2"), newLink("s2", "m1"), newLink("s2", "m2")},
		}},
		{"many_resolutions", PinState{
			Specs:   []Spec{newSpec("s1", "h1"), newSpec("s2", "h2")},
			Markers: []Marker{newMarker("m1", "h1"), newMarker("m2", "h2")},
			Links:   []Link{newLink("s1", "m1"), newLink("s2", "m2")},
			ResolutionState: []ResolutionState{
				newResolutionState("s1", "m1", "ch1", "ch2"),
				newResolutionState("s2", "m2", "ch3", "ch4"),
			},
		}},
		{"full_graph_3x3", PinState{
			Specs:   []Spec{newSpec("s1", "h1"), newSpec("s2", "h2"), newSpec("s3", "h3")},
			Markers: []Marker{newMarker("m1", "h1"), newMarker("m2", "h2"), newMarker("m3", "h3")},
			Links: []Link{
				newLink("s1", "m1"), newLink("s1", "m2"), newLink("s1", "m3"),
				newLink("s2", "m1"), newLink("s2", "m2"), newLink("s2", "m3"),
				newLink("s3", "m1"), newLink("s3", "m2"), newLink("s3", "m3"),
			},
			ResolutionState: []ResolutionState{
				newResolutionState("s1", "m1", "ch1", "ch2"),
				newResolutionState("s2", "m2", "ch3", "ch4"),
			},
		}},
		{"specs_with_locations", PinState{
			Specs: []Spec{
				newSpecWithLocation("s1", "h1", "/project/specs/auth.xml", 42),
				newSpecWithLocation("s2", "h2", "/project/specs/api.xml", 88),
			},
			Markers: []Marker{
				newMarkerWithLocation("m1", "h1", "/project/src/auth.go", 15),
				newMarkerWithLocation("m2", "h2", "/project/src/api.go", 200),
			},
			Links: []Link{newLink("s1", "m1"), newLink("s2", "m2")},
		}},
	}

	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			dir := t.TempDir()
			store := NewFilePinStore(dir)

			err := store.Save(shape.state)
			assertNoError(t, err)

			loaded, err := store.Load()
			assertNoError(t, err)
			assertPinStateEquals(t, loaded, shape.state)
		})
	}
}

func TestFilePinStoreLoadMissing(t *testing.T) {
	t.Run("missing_file_returns_error", func(t *testing.T) {
		dir := t.TempDir()
		store := NewFilePinStore(dir)
		_, err := store.Load()
		assertErrorWraps(t, err, ErrPinNotFound)
	})
}

func TestFilePinStoreLoadMalformed(t *testing.T) {
	t.Run("malformed_xml_returns_error", func(t *testing.T) {
		dir := t.TempDir()
		pinPath := filepath.Join(dir, "drift.pin")
		os.WriteFile(pinPath, []byte("not valid xml <"), 0644)
		store := NewFilePinStore(dir)
		_, err := store.Load()
		if err == nil {
			t.Fatalf("expected error loading malformed XML")
		}
	})
}

func TestFilePinStoreSaveOverwrite(t *testing.T) {
	t.Run("save_overwrites_existing", func(t *testing.T) {
		dir := t.TempDir()
		store := NewFilePinStore(dir)

		initial := PinState{
			Specs:   []Spec{newSpec("s1", "h1")},
			Markers: []Marker{newMarker("m1", "h1")},
			Links:   []Link{newLink("s1", "m1")},
		}
		err := store.Save(initial)
		assertNoError(t, err)

		err = store.Save(PinState{})
		assertNoError(t, err)

		loaded, err := store.Load()
		assertNoError(t, err)
		assertPinStateEquals(t, loaded, PinState{})
	})
}

func TestFilePinStoreSaveEmptyCreatesFile(t *testing.T) {
	t.Run("save_empty_creates_file", func(t *testing.T) {
		dir := t.TempDir()
		store := NewFilePinStore(dir)

		err := store.Save(PinState{})
		assertNoError(t, err)

		pinPath := filepath.Join(dir, "drift.pin")
		if _, err := os.Stat(pinPath); os.IsNotExist(err) {
			t.Fatalf("drift.pin not created")
		}
	})
}
