package scanner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"drift/internal/testutil"
	"drift/scanner"
)

func writeMainDrift(t *testing.T, dir, content string) {
	t.Helper()
	testutil.WriteSpecFile(t, dir, "main.drift.xml", content)
}

func writeModuleFile(t *testing.T, dir, name, content string) {
	t.Helper()
	testutil.WriteSpecFile(t, dir, name, content)
}

func assertScanError(t *testing.T, scanner *scanner.FileScanner, errContains string) {
	t.Helper()
	_, err := scanner.Scan()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", errContains)
	}
	if strings.Contains(err.Error(), errContains) {
		return
	}
	t.Fatalf("expected error containing %q, got %q", errContains, err.Error())
}

func TestScannerEmptyProject(t *testing.T) {
	t.Run("missing_main_pin_xml_errors", func(t *testing.T) {
		dir := t.TempDir()
		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "main.drift.xml")
	})

	t.Run("empty_main_returns_no_specs", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)
		if len(result.Specs) != 0 {
			t.Fatalf("expected 0 specs, got %d", len(result.Specs))
		}
	})

	t.Run("empty_main_still_discovers_markers", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("m1")+`
func a() {}
`+testutil.MarkerEnd("m1")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)
		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker, got %d", len(result.Markers))
		}
	})

	t.Run("specs_wrapper_rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <specs>
    <spec id="validate">input must be validated</spec>
  </specs>
</main>`)

		scanner := scanner.NewFileScanner(dir)
		_, err := scanner.Scan()
		if err == nil {
			t.Fatalf("expected error for <specs> wrapper, got nil")
		}
		if !strings.Contains(err.Error(), "<specs>") {
			t.Fatalf("error should mention <specs> wrapper, got: %s", err.Error())
		}
	})
}

func TestScannerSpecDiscovery(t *testing.T) {
	t.Run("main_with_direct_specs_implicit_main_module", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <spec id="validate_input">input must be validated</spec>
</main>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 1 {
			t.Fatalf("expected 1 spec, got %d", len(result.Specs))
		}
		spec, ok := testutil.FindScanResultSpec(result.Specs, "main.validate_input")
		if !ok {
			t.Fatalf("expected spec main.validate_input, not found. specs: %+v", result.Specs)
		}
		if spec.Filepath != "main.drift.xml" {
			t.Fatalf("filepath = %q, want %q", spec.Filepath, "main.drift.xml")
		}
		if spec.Hash == "" {
			t.Fatalf("expected non-empty hash")
		}
	})

	t.Run("main_imports_one_module", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./core.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "core.drift.xml", `<module name="core">
  <spec id="validate">Validation must reject duplicates.</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 1 {
			t.Fatalf("expected 1 spec, got %d", len(result.Specs))
		}
		spec, ok := testutil.FindScanResultSpec(result.Specs, "core.validate")
		if !ok {
			t.Fatalf("expected spec core.validate, not found. specs: %+v", result.Specs)
		}
		if spec.Filepath != "core.drift.xml" {
			t.Fatalf("filepath = %q, want %q", spec.Filepath, "core.drift.xml")
		}
	})

	t.Run("main_imports_multiple_modules", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./auth.drift.xml" />
  <import path="./api.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "auth.drift.xml", `<module name="auth">
  <spec id="login">Login required.</spec>
</module>`)
		writeModuleFile(t, dir, "api.drift.xml", `<module name="api">
  <spec id="endpoint">API endpoint must be versioned.</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 2 {
			t.Fatalf("expected 2 specs, got %d", len(result.Specs))
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "auth.login"); !ok {
			t.Fatalf("expected spec auth.login, not found")
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "api.endpoint"); !ok {
			t.Fatalf("expected spec api.endpoint, not found")
		}
	})

	t.Run("main_with_direct_specs_and_imports", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./core.drift.xml" />
  <spec id="app_entry">App entry point must validate config.</spec>
</main>`)
		writeModuleFile(t, dir, "core.drift.xml", `<module name="core">
  <spec id="validate">Validation must reject duplicates.</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 2 {
			t.Fatalf("expected 2 specs, got %d", len(result.Specs))
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "main.app_entry"); !ok {
			t.Fatalf("expected spec main.app_entry, not found")
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "core.validate"); !ok {
			t.Fatalf("expected spec core.validate, not found")
		}
	})

	t.Run("one_module_many_specs", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./core.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "core.drift.xml", `<module name="core">
  <spec id="validate">validate spec</spec>
  <spec id="authenticate">auth spec</spec>
  <spec id="log">log spec</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 3 {
			t.Fatalf("expected 3 specs, got %d", len(result.Specs))
		}
		for _, id := range []string{"core.validate", "core.authenticate", "core.log"} {
			if _, ok := testutil.FindScanResultSpec(result.Specs, id); !ok {
				t.Fatalf("expected spec %s, not found", id)
			}
		}
	})

	t.Run("spec_missing_id_attribute_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <spec>no id here</spec>
</main>`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "id")
	})

	t.Run("duplicate_spec_ids_within_same_module_error", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <spec id="dup">first</spec>
  <spec id="dup">second</spec>
</main>`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "duplicate")
	})

	t.Run("same_spec_id_in_different_modules_ok", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./a.drift.xml" />
  <import path="./b.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "a.drift.xml", `<module name="a">
  <spec id="shared">a version</spec>
</module>`)
		writeModuleFile(t, dir, "b.drift.xml", `<module name="b">
  <spec id="shared">b version</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 2 {
			t.Fatalf("expected 2 specs, got %d", len(result.Specs))
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "a.shared"); !ok {
			t.Fatalf("expected spec a.shared, not found")
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "b.shared"); !ok {
			t.Fatalf("expected spec b.shared, not found")
		}
	})

	t.Run("hash_is_sha1_deterministic", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <spec id="s1">deterministic content</spec>
</main>`)

		scanner := scanner.NewFileScanner(dir)
		result1, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		result2, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		spec1, _ := testutil.FindScanResultSpec(result1.Specs, "main.s1")
		spec2, _ := testutil.FindScanResultSpec(result2.Specs, "main.s1")

		if spec1.Hash != spec2.Hash {
			t.Fatalf("hash not deterministic: %q vs %q", spec1.Hash, spec2.Hash)
		}
	})
}

func TestScannerImportGraph(t *testing.T) {
	t.Run("transitive_imports_all_loaded", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./a.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "a.drift.xml", `<module name="a">
  <import path="./b.drift.xml" />
  <spec id="spec_a">a spec</spec>
</module>`)
		writeModuleFile(t, dir, "b.drift.xml", `<module name="b">
  <spec id="spec_b">b spec</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 2 {
			t.Fatalf("expected 2 specs, got %d", len(result.Specs))
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "a.spec_a"); !ok {
			t.Fatalf("expected spec a.spec_a, not found")
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "b.spec_b"); !ok {
			t.Fatalf("expected spec b.spec_b, not found")
		}
	})

	t.Run("diamond_imports_deduplicated", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./a.drift.xml" />
  <import path="./b.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "a.drift.xml", `<module name="a">
  <import path="./shared.drift.xml" />
  <spec id="spec_a">a spec</spec>
</module>`)
		writeModuleFile(t, dir, "b.drift.xml", `<module name="b">
  <import path="./shared.drift.xml" />
  <spec id="spec_b">b spec</spec>
</module>`)
		writeModuleFile(t, dir, "shared.drift.xml", `<module name="shared">
  <spec id="spec_shared">shared spec</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 3 {
			t.Fatalf("expected 3 specs (shared loaded once), got %d: %+v", len(result.Specs), result.Specs)
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "a.spec_a"); !ok {
			t.Fatalf("expected spec a.spec_a, not found")
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "b.spec_b"); !ok {
			t.Fatalf("expected spec b.spec_b, not found")
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "shared.spec_shared"); !ok {
			t.Fatalf("expected spec shared.spec_shared, not found")
		}
	})

	t.Run("duplicate_module_names_error", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./a.drift.xml" />
  <import path="./b.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "a.drift.xml", `<module name="dup">
  <spec id="spec_a">a spec</spec>
</module>`)
		writeModuleFile(t, dir, "b.drift.xml", `<module name="dup">
  <spec id="spec_b">b spec</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "duplicate module")
	})

	t.Run("cycle_detection_errors_with_trace", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./a.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "a.drift.xml", `<module name="a">
  <import path="./b.drift.xml" />
  <spec id="spec_a">a spec</spec>
</module>`)
		writeModuleFile(t, dir, "b.drift.xml", `<module name="b">
  <import path="./a.drift.xml" />
  <spec id="spec_b">b spec</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "cycle")
	})

	t.Run("import_path_not_found_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./nonexistent.drift.xml" />
</main>`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "nonexistent.drift.xml")
	})

	t.Run("imports_in_subdirectories", func(t *testing.T) {
		dir := t.TempDir()
		subdir := filepath.Join(dir, "sub")
		os.Mkdir(subdir, 0755)
		writeMainDrift(t, dir, `<main>
  <import path="./sub/nested.drift.xml" />
</main>`)
		writeModuleFile(t, subdir, "nested.drift.xml", `<module name="nested">
  <spec id="deep">deep spec</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 1 {
			t.Fatalf("expected 1 spec, got %d", len(result.Specs))
		}
		if _, ok := testutil.FindScanResultSpec(result.Specs, "nested.deep"); !ok {
			t.Fatalf("expected spec nested.deep, not found")
		}
	})

	t.Run("module_without_name_attribute_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./core.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "core.drift.xml", `<module>
  <spec id="validate">validate spec</spec>
</module>`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "name")
	})
}

func TestScannerMarkerDiscovery(t *testing.T) {
	t.Run("one_code_file_one_marker", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", `package main

`+testutil.MarkerStart("abc123")+`
func handleRequest() {
	doSomething()
}
`+testutil.MarkerEnd("abc123")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker, got %d", len(result.Markers))
		}
		marker, ok := testutil.FindScanResultMarker(result.Markers, "abc123")
		if !ok {
			t.Fatalf("expected marker abc123, not found")
		}
		if marker.Filepath != "main.go" {
			t.Fatalf("filepath = %q, want %q", marker.Filepath, "main.go")
		}
		if marker.Hash == "" {
			t.Fatalf("expected non-empty hash")
		}
		if marker.LineNumber != 3 {
			t.Fatalf("LineNumber = %d, want 3", marker.LineNumber)
		}
		if marker.EndLineNumber != 7 {
			t.Fatalf("EndLineNumber = %d, want 7", marker.EndLineNumber)
		}
	})

	t.Run("one_code_file_many_markers", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", `package main

`+testutil.MarkerStart("m1")+`
func handlerA() {
	a()
}
`+testutil.MarkerEnd("m1")+`

`+testutil.MarkerStart("m2")+`
func handlerB() {
	b()
}
`+testutil.MarkerEnd("m2")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers, got %d", len(result.Markers))
		}
		for _, id := range []string{"m1", "m2"} {
			if _, ok := testutil.FindScanResultMarker(result.Markers, id); !ok {
				t.Fatalf("expected marker %s, not found", id)
			}
		}
	})

	t.Run("many_code_files", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "a.go", testutil.MarkerStart("ma")+`
func a() { x() }
`+testutil.MarkerEnd("ma")+`
`)
		testutil.WriteCodeFile(t, dir, "b.go", testutil.MarkerStart("mb")+`
func b() { y() }
`+testutil.MarkerEnd("mb")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers, got %d", len(result.Markers))
		}
		for _, id := range []string{"ma", "mb"} {
			if _, ok := testutil.FindScanResultMarker(result.Markers, id); !ok {
				t.Fatalf("expected marker %s, not found", id)
			}
		}
	})

	t.Run("duplicate_marker_shortcodes_error", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "a.go", testutil.MarkerStart("dup")+`
func a() { }
`+testutil.MarkerEnd("dup")+`
`)
		testutil.WriteCodeFile(t, dir, "b.go", testutil.MarkerStart("dup")+`
func b() { }
`+testutil.MarkerEnd("dup")+`
`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "duplicate marker")
	})

	t.Run("marker_hash_is_sha1_deterministic", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("abc")+`
func handler() {
	doSomething()
}
`+testutil.MarkerEnd("abc")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result1, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		result2, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		m1, _ := testutil.FindScanResultMarker(result1.Markers, "abc")
		m2, _ := testutil.FindScanResultMarker(result2.Markers, "abc")

		if m1.Hash != m2.Hash {
			t.Fatalf("hash not deterministic: %q vs %q", m1.Hash, m2.Hash)
		}
	})
}

func TestScannerRangeHashing(t *testing.T) {
	t.Run("hashes_lines_between_start_and_end_exclusive", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		code := testutil.MarkerStart("abc") + `
line2
line3
line4
` + testutil.MarkerEnd("abc") + `
line6
`
		testutil.WriteCodeFile(t, dir, "main.go", code)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		marker, _ := testutil.FindScanResultMarker(result.Markers, "abc")

		expectedContent := `line2
line3
line4
`
		expectedHash := testutil.ExpectedSha1Hex(expectedContent)
		if marker.Hash != expectedHash {
			t.Fatalf("hash = %q, want %q (content hash of lines between start and end)", marker.Hash, expectedHash)
		}
	})

	t.Run("empty_range_hashes_empty_string", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		code := testutil.MarkerStart("abc") + `
` + testutil.MarkerEnd("abc") + `
`
		testutil.WriteCodeFile(t, dir, "main.go", code)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		marker, _ := testutil.FindScanResultMarker(result.Markers, "abc")

		expectedHash := testutil.ExpectedSha1Hex("")
		if marker.Hash != expectedHash {
			t.Fatalf("hash = %q, want %q (empty range)", marker.Hash, expectedHash)
		}
	})

	t.Run("stores_start_and_end_line_numbers", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		code := `package main

` + testutil.MarkerStart("abc") + `
func handler() {
	doSomething()
}
` + testutil.MarkerEnd("abc") + `
`
		testutil.WriteCodeFile(t, dir, "main.go", code)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		marker, _ := testutil.FindScanResultMarker(result.Markers, "abc")
		if marker.LineNumber != 3 {
			t.Fatalf("LineNumber = %d, want 3", marker.LineNumber)
		}
		if marker.EndLineNumber != 7 {
			t.Fatalf("EndLineNumber = %d, want 7", marker.EndLineNumber)
		}
	})

	t.Run("single_line_range_hashes_single_content_line", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		code := testutil.MarkerStart("abc") + `
only_line
` + testutil.MarkerEnd("abc") + `
`
		testutil.WriteCodeFile(t, dir, "main.go", code)

		sc := scanner.NewFileScanner(dir)
		result, err := sc.Scan()
		testutil.AssertNoError(t, err)

		marker, _ := testutil.FindScanResultMarker(result.Markers, "abc")

		expectedHash := testutil.ExpectedSha1Hex("only_line\n")
		if marker.Hash != expectedHash {
			t.Fatalf("hash = %q, want %q (single content line)", marker.Hash, expectedHash)
		}
	})
}

func TestScannerMixedSpecsAndMarkers(t *testing.T) {
	t.Run("specs_and_markers_across_multiple_files", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <import path="./specs.drift.xml" />
</main>`)
		writeModuleFile(t, dir, "specs.drift.xml", `<module name="specs">
  <spec id="validate_input">input must be validated</spec>
  <spec id="auth_check">auth must be checked</spec>
</module>`)
		testutil.WriteCodeFile(t, dir, "auth.go", testutil.MarkerStart("m1")+`
func auth() { check() }
`+testutil.MarkerEnd("m1")+`
`)
		testutil.WriteCodeFile(t, dir, "input.go", testutil.MarkerStart("m2")+`
func validate() { check() }
`+testutil.MarkerEnd("m2")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Specs) != 2 {
			t.Fatalf("expected 2 specs, got %d", len(result.Specs))
		}
		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers, got %d", len(result.Markers))
		}
	})
}

func TestScannerDriftIgnore(t *testing.T) {
	t.Run("no_drift_ignore_scans_all", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("keep")+`
func a() {}
`+testutil.MarkerEnd("keep")+`
`)
		testutil.WriteCodeFile(t, dir, "main_test.go", testutil.MarkerStart("drop")+`
func b() {}
`+testutil.MarkerEnd("drop")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers without drift.ignore, got %d", len(result.Markers))
		}
	})

	t.Run("star_test_go_excludes_test_files", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteIgnoreFile(t, dir, "*_test.go\n")
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("keep")+`
func a() {}
`+testutil.MarkerEnd("keep")+`
`)
		testutil.WriteCodeFile(t, dir, "main_test.go", testutil.MarkerStart("drop")+`
func b() {}
`+testutil.MarkerEnd("drop")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker, got %d", len(result.Markers))
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "keep"); !ok {
			t.Fatalf("expected marker 'keep', not found")
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "drop"); ok {
			t.Fatalf("marker 'drop' should have been excluded")
		}
	})

	t.Run("trailing_slash_skips_directory_subtree", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteIgnoreFile(t, dir, ".git/\n")
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("keep")+`
func a() {}
`+testutil.MarkerEnd("keep")+`
`)
		gitDir := filepath.Join(dir, ".git")
		os.Mkdir(gitDir, 0755)
		testutil.WriteCodeFile(t, gitDir, "hook.go", testutil.MarkerStart("drop")+`
func b() {}
`+testutil.MarkerEnd("drop")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker, got %d", len(result.Markers))
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "keep"); !ok {
			t.Fatalf("expected marker 'keep', not found")
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "drop"); ok {
			t.Fatalf("marker 'drop' from .git/ should have been excluded")
		}
	})

	t.Run("comments_and_empty_lines_ignored", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteIgnoreFile(t, dir, "# this is a comment\n\n*_test.go\n# another comment\n")
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("keep")+`
func a() {}
`+testutil.MarkerEnd("keep")+`
`)
		testutil.WriteCodeFile(t, dir, "main_test.go", testutil.MarkerStart("drop")+`
func b() {}
`+testutil.MarkerEnd("drop")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker, got %d", len(result.Markers))
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "keep"); !ok {
			t.Fatalf("expected marker 'keep', not found")
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "drop"); ok {
			t.Fatalf("marker 'drop' should have been excluded")
		}
	})

	t.Run("path_pattern_excludes_specific_file", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteIgnoreFile(t, dir, "sub/skip.go\n")
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("keep")+`
func a() {}
`+testutil.MarkerEnd("keep")+`
`)
		subDir := filepath.Join(dir, "sub")
		os.Mkdir(subDir, 0755)
		testutil.WriteCodeFile(t, subDir, "skip.go", testutil.MarkerStart("drop")+`
func b() {}
`+testutil.MarkerEnd("drop")+`
`)
		testutil.WriteCodeFile(t, subDir, "keep.go", testutil.MarkerStart("also_keep")+`
func c() {}
`+testutil.MarkerEnd("also_keep")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers, got %d", len(result.Markers))
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "drop"); ok {
			t.Fatalf("marker 'drop' should have been excluded by path pattern")
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "keep"); !ok {
			t.Fatalf("expected marker 'keep', not found")
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "also_keep"); !ok {
			t.Fatalf("expected marker 'also_keep', not found")
		}
	})
}

func TestScannerIgnoresNonPinXmlNonCodeFiles(t *testing.T) {
	t.Run("ignores_txt_md_json_files", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "notes.txt", testutil.MarkerStart("should_not_find")+"\n")
		testutil.WriteCodeFile(t, dir, "readme.md", testutil.MarkerStart("should_not_find_either")+"\n")
		testutil.WriteCodeFile(t, dir, "data.json", testutil.MarkerStart("nope")+"\n")

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 0 {
			t.Fatalf("expected 0 markers from non-code files, got %d", len(result.Markers))
		}
		if len(result.Specs) != 0 {
			t.Fatalf("expected 0 specs, got %d", len(result.Specs))
		}
	})
}

func TestScannerIDFormatValidation(t *testing.T) {
	t.Run("spec_id_with_dot_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <spec id="bad.id">spec with dot in id</spec>
</main>`)
		sc := scanner.NewFileScanner(dir)
		assertScanError(t, sc, "must not contain a dot")
	})

	t.Run("marker_id_with_dot_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <spec id="s1">spec</spec>
</main>`)
		badMarker := "// D! id=bad" + ".marker range-start\nfunc a() {}\n// D! id=bad" + ".marker range-end\n"
		testutil.WriteCodeFile(t, dir, "main.go", badMarker)
		sc := scanner.NewFileScanner(dir)
		assertScanError(t, sc, "must not contain a dot")
	})

	t.Run("spec_id_without_dot_ok", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main>
  <spec id="good_id">spec without dot</spec>
</main>`)
		sc := scanner.NewFileScanner(dir)
		result, err := sc.Scan()
		testutil.AssertNoError(t, err)
		if len(result.Specs) != 1 {
			t.Fatalf("expected 1 spec, got %d", len(result.Specs))
		}
		if result.Specs[0].ID != "main.good_id" {
			t.Fatalf("expected main.good_id, got %q", result.Specs[0].ID)
		}
	})

	t.Run("marker_id_without_dot_ok", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("good_marker")+`
func a() {}
`+testutil.MarkerEnd("good_marker")+`
`)
		sc := scanner.NewFileScanner(dir)
		result, err := sc.Scan()
		testutil.AssertNoError(t, err)
		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker, got %d", len(result.Markers))
		}
		if result.Markers[0].ID != "good_marker" {
			t.Fatalf("expected good_marker, got %q", result.Markers[0].ID)
		}
	})
}

func TestScannerRangeMarkers(t *testing.T) {
	t.Run("old_style_marker_without_suffix_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", "// D! id=foo\nfunc a() {}\n")

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "range-start")
	})

	t.Run("range_start_without_range_end_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("foo")+`
func a() {}
`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "range-start")
		assertScanError(t, scanner, "foo")
	})

	t.Run("range_end_without_range_start_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", `package main

`+testutil.MarkerEnd("foo")+`
`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "range-end")
		assertScanError(t, scanner, "foo")
	})

	t.Run("all_unpaired_markers_reported_at_once", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("foo")+`
func a() {}
`+testutil.MarkerEnd("bar")+`
`)

		scanner := scanner.NewFileScanner(dir)
		_, err := scanner.Scan()
		if err == nil {
			t.Fatalf("expected error for unpaired markers, got nil")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "foo") {
			t.Fatalf("error should mention 'foo', got: %s", errStr)
		}
		if !strings.Contains(errStr, "bar") {
			t.Fatalf("error should mention 'bar', got: %s", errStr)
		}
	})

	t.Run("range_end_before_range_start_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", `package main

`+testutil.MarkerEnd("foo")+`
func a() {}
`+testutil.MarkerStart("foo")+`
`)

		scanner := scanner.NewFileScanner(dir)
		assertScanError(t, scanner, "foo")
	})

	t.Run("nested_ranges_allowed", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("outer")+`
func a() {
`+testutil.MarkerStart("inner")+`
	b()
`+testutil.MarkerEnd("inner")+`
}
`+testutil.MarkerEnd("outer")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers, got %d", len(result.Markers))
		}
	})

	t.Run("overlapping_ranges_allowed", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("a")+`
line1
`+testutil.MarkerStart("b")+`
line2
`+testutil.MarkerEnd("a")+`
line3
`+testutil.MarkerEnd("b")+`
`)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers, got %d", len(result.Markers))
		}
	})

	t.Run("duplicate_range_start_for_same_id_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("x")+`
content1
`+testutil.MarkerStart("x")+`
content2
`+testutil.MarkerEnd("x")+`
`)

		sc := scanner.NewFileScanner(dir)
		assertScanError(t, sc, "duplicate range-start")
	})

	t.Run("duplicate_range_end_for_same_id_errors", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("x")+`
content1
`+testutil.MarkerEnd("x")+`
content2
`+testutil.MarkerEnd("x")+`
`)

		sc := scanner.NewFileScanner(dir)
		assertScanError(t, sc, "duplicate range-end")
	})

	t.Run("multiple_unpaired_starts_reported_at_once", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "main.go", testutil.MarkerStart("aaa")+`
content_a
`+testutil.MarkerStart("bbb")+`
content_b
`)

		sc := scanner.NewFileScanner(dir)
		_, err := sc.Scan()
		if err == nil {
			t.Fatalf("expected error for unpaired markers, got nil")
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "aaa") {
			t.Fatalf("error should mention 'aaa', got: %s", errStr)
		}
		if !strings.Contains(errStr, "bbb") {
			t.Fatalf("error should mention 'bbb', got: %s", errStr)
		}
	})
}

func TestScannerMarkerBlanking(t *testing.T) {
	t.Run("other_marker_declarations_blanked_from_hash", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		code := testutil.MarkerStart("outer") + `
func a() {
` + testutil.MarkerStart("inner") + `
	b()
` + testutil.MarkerEnd("inner") + `
}
` + testutil.MarkerEnd("outer") + `
`
		testutil.WriteCodeFile(t, dir, "main.go", code)

		scanner := scanner.NewFileScanner(dir)
		result, err := scanner.Scan()
		testutil.AssertNoError(t, err)

		outer, _ := testutil.FindScanResultMarker(result.Markers, "outer")

		expectedContent := `func a() {
// 
	b()
// 
}
`
		expectedHash := testutil.ExpectedSha1Hex(expectedContent)
		if outer.Hash != expectedHash {
			t.Fatalf("outer hash = %q, want %q (marker declarations blanked)\nexpected content:\n%q", outer.Hash, expectedHash, expectedContent)
		}
	})

	t.Run("changing_inner_marker_id_does_not_change_outer_hash", func(t *testing.T) {
		dir1 := t.TempDir()
		writeMainDrift(t, dir1, `<main></main>`)
		code1 := testutil.MarkerStart("outer") + `
func a() {
` + testutil.MarkerStart("inner_a") + `
	b()
` + testutil.MarkerEnd("inner_a") + `
}
` + testutil.MarkerEnd("outer") + `
`
		testutil.WriteCodeFile(t, dir1, "main.go", code1)

		dir2 := t.TempDir()
		writeMainDrift(t, dir2, `<main></main>`)
		code2 := testutil.MarkerStart("outer") + `
func a() {
` + testutil.MarkerStart("inner_b") + `
	b()
` + testutil.MarkerEnd("inner_b") + `
}
` + testutil.MarkerEnd("outer") + `
`
		testutil.WriteCodeFile(t, dir2, "main.go", code2)

		sc1 := scanner.NewFileScanner(dir1)
		r1, err := sc1.Scan()
		testutil.AssertNoError(t, err)

		sc2 := scanner.NewFileScanner(dir2)
		r2, err := sc2.Scan()
		testutil.AssertNoError(t, err)

		m1, _ := testutil.FindScanResultMarker(r1.Markers, "outer")
		m2, _ := testutil.FindScanResultMarker(r2.Markers, "outer")

		if m1.Hash != m2.Hash {
			t.Fatalf("outer hash should be the same regardless of inner marker ID\ninner_a: %q\ninner_b: %q", m1.Hash, m2.Hash)
		}
	})

	t.Run("overlapping_ranges_blank_inner_marker_declarations", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		// A: lines 1-10, B: lines 5-15 (overlapping, not nested)
		code := testutil.MarkerStart("A") + `
line2
line3
line4
` + testutil.MarkerStart("B") + `
line6
line7
line8
line9
` + testutil.MarkerEnd("A") + `
line11
line12
line13
line14
` + testutil.MarkerEnd("B") + `
`
		testutil.WriteCodeFile(t, dir, "main.go", code)

		sc := scanner.NewFileScanner(dir)
		result, err := sc.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 2 {
			t.Fatalf("expected 2 markers, got %d", len(result.Markers))
		}

		markerA, ok := testutil.FindScanResultMarker(result.Markers, "A")
		if !ok {
			t.Fatalf("marker A not found")
		}
		markerB, ok := testutil.FindScanResultMarker(result.Markers, "B")
		if !ok {
			t.Fatalf("marker B not found")
		}

		// A's hash: content between A's start and end, with B's declaration blanked
		expectedAContent := "line2\nline3\nline4\n// \nline6\nline7\nline8\nline9\n"
		expectedAHash := testutil.ExpectedSha1Hex(expectedAContent)
		if markerA.Hash != expectedAHash {
			t.Fatalf("A hash = %q, want %q", markerA.Hash, expectedAHash)
		}

		// B's hash: content between B's start and end, with A's end blanked
		expectedBContent := "line6\nline7\nline8\nline9\n// \nline11\nline12\nline13\nline14\n"
		expectedBHash := testutil.ExpectedSha1Hex(expectedBContent)
		if markerB.Hash != expectedBHash {
			t.Fatalf("B hash = %q, want %q", markerB.Hash, expectedBHash)
		}
	})
}

func TestScannerNonGoExtensions(t *testing.T) {
	t.Run("py_file_markers_discovered", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "script.py", "# D! id=py1 range-start\ndef hello():\n    pass\n# D! id=py1 range-end\n")

		sc := scanner.NewFileScanner(dir)
		result, err := sc.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker from .py file, got %d", len(result.Markers))
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "py1"); !ok {
			t.Fatalf("expected marker py1, not found")
		}
	})

	t.Run("js_file_markers_discovered", func(t *testing.T) {
		dir := t.TempDir()
		writeMainDrift(t, dir, `<main></main>`)
		testutil.WriteCodeFile(t, dir, "app.js", "// D! id=js1 range-start\nfunction hello() {\n  return 1;\n}\n// D! id=js1 range-end\n")

		sc := scanner.NewFileScanner(dir)
		result, err := sc.Scan()
		testutil.AssertNoError(t, err)

		if len(result.Markers) != 1 {
			t.Fatalf("expected 1 marker from .js file, got %d", len(result.Markers))
		}
		if _, ok := testutil.FindScanResultMarker(result.Markers, "js1"); !ok {
			t.Fatalf("expected marker js1, not found")
		}
	})
}

func TestScannerSpecErrors(t *testing.T) {
	t.Run("malformed_xml_errors", func(t *testing.T) {
		dir := t.TempDir()
		testutil.WriteSpecFile(t, dir, "main.drift.xml", `<<invalid>>`)

		sc := scanner.NewFileScanner(dir)
		_, err := sc.Scan()
		if err == nil {
			t.Fatalf("expected error for malformed XML, got nil")
		}
		if !strings.Contains(err.Error(), "xml") {
			t.Fatalf("error should mention xml, got: %s", err.Error())
		}
	})

	t.Run("wrong_root_element_errors", func(t *testing.T) {
		dir := t.TempDir()
		testutil.WriteSpecFile(t, dir, "main.drift.xml", `<foo><spec id="a">text</spec></foo>`)

		sc := scanner.NewFileScanner(dir)
		assertScanError(t, sc, "expected <main> or <module>")
	})
}
