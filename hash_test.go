package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestHashAlgorithm(t *testing.T) {
	// #F id:8y5qunwi hash.algorithm
	spec, _ := mustParse(t, "testdata/golden.spec.xml")
	hashes := ComputeAllHashes(spec)

	h := sha256.Sum256([]byte("a"))
	want := hex.EncodeToString(h[:])
	if got := hashes["x"]; got != want {
		t.Errorf("hash(x) = %s, want %s", got, want)
	}
}

func TestHashInput_Content(t *testing.T) {
	// #F id:ih9ocp56 hash.input.content
	spec, _ := mustParse(t, "testdata/golden.spec.xml")

	if got := findClause(spec, "x").Content(); got != "a" {
		t.Errorf("x.Content() = %q, want %q", got, "a")
	}
}

func TestHashInput_References(t *testing.T) {
	// #F id:ljsysabe hash.input.references
	spec, _ := mustParse(t, "testdata/golden.spec.xml")
	hashes := ComputeAllHashes(spec)

	xRaw := sha256.Sum256([]byte("a"))
	xHashHex := hex.EncodeToString(xRaw[:])
	h := sha256.New()
	h.Write([]byte("b cites x."))
	h.Write([]byte{0x0A})
	h.Write([]byte(xHashHex))
	want := hex.EncodeToString(h.Sum(nil))
	if got := hashes["y"]; got != want {
		t.Errorf("hash(y) = %s, want %s", got, want)
	}
}

func TestHashInput_ReferencesOrder(t *testing.T) {
	// #F id:vksy9yvg hash.input.references_order
	spec, _ := mustParse(t, "testdata/golden.spec.xml")
	defined := spec.DefinedIDs()

	z := findClause(spec, "z")
	refs := ReferencesInOrder(z, defined)
	if len(refs) != 2 || refs[0] != "y" || refs[1] != "x" {
		t.Errorf("ReferencesInOrder(z) = %v, want [y x]", refs)
	}
}

func TestHashInput_Whitespace(t *testing.T) {
	// #F id:nq4x94t2 hash.input.whitespace
	spec, _ := mustParse(t, "testdata/golden.spec.xml")

	if got := findClause(spec, "x").Content(); got != "a" {
		t.Errorf("x.Content() = %q, want %q (trimmed)", got, "a")
	}
}

func TestHashInput_Separator(t *testing.T) {
	// #F id:tj2bb4lk hash.input.separator
	h := sha256.New()
	h.Write([]byte("content"))
	h.Write([]byte{0x0A})
	h.Write([]byte("aaa"))
	h.Write([]byte{0x0A})
	h.Write([]byte("bbb"))
	want := hex.EncodeToString(h.Sum(nil))

	got := hashOf("content", []string{"aaa", "bbb"})
	if got != want {
		t.Errorf("hashOf with separators = %s, want %s", got, want)
	}
}

func TestHashOutput_Format(t *testing.T) {
	// #F id:4aulhk25 hash.output.format
	spec, _ := mustParse(t, "testdata/golden.spec.xml")
	hashes := ComputeAllHashes(spec)

	for path, h := range hashes {
		if len(h) != 64 {
			t.Errorf("hash(%s) has length %d, want 64", path, len(h))
		}
		for i := 0; i < len(h); i++ {
			c := h[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("hash(%s) contains non-hex char %q at position %d", path, c, i)
			}
		}
	}
}

func findClause(spec *Spec, id string) *Element {
	for _, e := range spec.All() {
		if e.ID == id {
			return e
		}
	}
	return nil
}
