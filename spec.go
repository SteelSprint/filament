package main

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// #F id:3aewemki hash.algorithm hash.input.content hash.input.references hash.input.references_order hash.input.whitespace hash.input.separator hash.output.format path_format.structure path_format.charset path_format.depth










type ElementKind int

const (
	KindClause ElementKind = iota
	KindSection
	KindTerm
)

func (k ElementKind) String() string {
	switch k {
	case KindClause:
		return "clause"
	case KindSection:
		return "section"
	case KindTerm:
		return "term"
	}
	return "unknown"
}

type Element struct {
	Kind          ElementKind
	ID            string
	Label         string
	Text          string
	Parent        *Element
	Kids          []*Element
	InDefinitions bool
}

func (e *Element) Path() string {
	return e.ID
}

type Spec struct {
	Name        string
	Description string
	Elements    []*Element
}

func (s *Spec) All() []*Element {
	var out []*Element
	var walk func(e *Element)
	walk = func(e *Element) {
		for _, k := range e.Kids {
			out = append(out, k)
			walk(k)
		}
	}
	walk(&Element{Kids: s.Elements})
	return out
}

func (s *Spec) DefinedIDs() map[string]bool {
	out := make(map[string]bool)
	for _, e := range s.All() {
		if isStructural(e) {
			continue
		}
		out[e.ID] = true
	}
	return out
}

func isStructural(e *Element) bool {
	if e.Kind == KindSection && (e.ID == "spec" || e.ID == "description" || e.ID == "definitions") {
		return true
	}
	return false
}

func (e *Element) HasText() bool {
	t := e.Text
	for i := 0; i < len(t); i++ {
		c := t[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return true
		}
	}
	return false
}

func (e *Element) Content() string {
	if e.Kind == KindSection {
		return e.Label
	}
	return trimSpace(e.Text)
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

type tokenPos struct {
	offset int
	tok    string
}

func isRefChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '.'
}

func stripTrailingNonID(s string) string {
	end := len(s)
	for end > 0 {
		c := s[end-1]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			break
		}
		end--
	}
	return s[:end]
}

func findRefTokens(text string) []tokenPos {
	var out []tokenPos
	i := 0
	for i < len(text) {
		if isRefChar(text[i]) {
			start := i
			for i < len(text) && isRefChar(text[i]) {
				i++
			}
			tok := text[start:i]
			tok = stripTrailingNonID(tok)
			if tok != "" {
				out = append(out, tokenPos{start, tok})
			}
		} else {
			i++
		}
	}
	return out
}

func ReferencesInOrder(e *Element, defined map[string]bool) []string {
	if e.Kind == KindSection {
		return nil
	}
	tokens := findRefTokens(e.Text)
	type pos struct {
		offset int
		id     string
	}
	var found []pos
	seen := make(map[string]bool)
	for _, tp := range tokens {
		if !defined[tp.tok] {
			continue
		}
		if seen[tp.tok] {
			continue
		}
		seen[tp.tok] = true
		found = append(found, pos{tp.offset, tp.tok})
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].offset != found[j].offset {
			return found[i].offset < found[j].offset
		}
		return found[i].id < found[j].id
	})
	out := make([]string, 0, len(found))
	for _, p := range found {
		out = append(out, p.id)
	}
	return out
}

func ComputeHash(e *Element, defined map[string]bool, hashes map[string]string) string {
	if e.Kind == KindSection {
		return hashOf(e.Content(), nil)
	}
	refs := ReferencesInOrder(e, defined)
	hashRefs := make([]string, 0, len(refs))
	for _, r := range refs {
		hashRefs = append(hashRefs, hashes[r])
	}
	return hashOf(e.Content(), hashRefs)
}

func hashOf(content string, refs []string) string {
	h := sha256.New()
	h.Write([]byte(content))
	for _, r := range refs {
		h.Write([]byte{0x0A})
		h.Write([]byte(r))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func ComputeAllHashes(s *Spec) map[string]string {
	hashes := make(map[string]string)
	defined := s.DefinedIDs()
	for _, e := range s.All() {
		hashes[e.Path()] = ComputeHash(e, defined, hashes)
	}
	return hashes
}
