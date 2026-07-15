package driftpin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInit(t *testing.T) {
	t.Run("init_creates_drift_pin", func(t *testing.T) {
		dir := t.TempDir()
		output, code := Run([]string{"init"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0, output: %s", code, output)
		}
		pinPath := filepath.Join(dir, "drift.pin")
		if _, err := os.Stat(pinPath); os.IsNotExist(err) {
			t.Fatalf("drift.pin not created")
		}
	})

	t.Run("init_then_todo_no_changes", func(t *testing.T) {
		dir := t.TempDir()

		_, code := Run([]string{"init"}, dir)
		if code != 0 {
			t.Fatalf("init failed with non-zero exit code")
		}

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0, output: %s", code, output)
		}
		expected := "No changes detected."
		if output != expected {
			t.Fatalf("output = %q, want %q", output, expected)
		}
	})

	t.Run("init_creates_valid_empty_pin", func(t *testing.T) {
		dir := t.TempDir()
		_, code := Run([]string{"init"}, dir)
		if code != 0 {
			t.Fatalf("init failed")
		}

		store := NewFilePinStore(dir)
		state, err := store.Load()
		assertNoError(t, err)
		assertPinStateEquals(t, state, PinState{})
	})
}

func TestCLITodoWithoutInit(t *testing.T) {
	t.Run("todo_without_init_errors", func(t *testing.T) {
		dir := t.TempDir()
		output, code := Run([]string{"todo"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
		if !strings.Contains(strings.ToLower(output), "init") {
			t.Fatalf("error message should mention init, got: %s", output)
		}
	})
}

func TestCLIResetWithoutInit(t *testing.T) {
	t.Run("reset_without_init_errors", func(t *testing.T) {
		dir := t.TempDir()
		output, code := Run([]string{"reset", "m1:s1"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
	})
}

func TestCLINoArgs(t *testing.T) {
	t.Run("no_args_shows_usage", func(t *testing.T) {
		dir := t.TempDir()
		output, code := Run([]string{}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code for no args")
		}
		if !strings.Contains(strings.ToLower(output), "usage") {
			t.Fatalf("output should contain usage, got: %s", output)
		}
	})
}

func TestCLIUnknownCommand(t *testing.T) {
	t.Run("unknown_command_errors", func(t *testing.T) {
		dir := t.TempDir()
		output, code := Run([]string{"frobnicate"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code for unknown command")
		}
		if !strings.Contains(strings.ToLower(output), "unknown") {
			t.Fatalf("output should mention unknown command, got: %s", output)
		}
	})
}

func TestCLIResetBadFormat(t *testing.T) {
	t.Run("reset_without_argument", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)
		output, code := Run([]string{"reset"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
	})

	t.Run("reset_bad_format_no_colon", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)
		output, code := Run([]string{"reset", "no_colon_here"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
	})
}

func TestCLIFullFlowSpecMarkerLinkDrift(t *testing.T) {
	t.Run("init_create_spec_create_marker_todo_no_links_no_drift", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="validate_input">input must be validated</spec></specs>`)
		writeCodeFile(t, dir, "main.go", `// #F abc123
func handleRequest() {
	doSomething()
}
`)

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0, output: %s", code, output)
		}
		if output != "No changes detected." {
			t.Fatalf("output = %q, want %q", output, "No changes detected.")
		}
	})

	t.Run("link_then_todo_no_drift", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="validate_input">input must be validated</spec></specs>`)
		writeCodeFile(t, dir, "main.go", `// #F abc123
func handleRequest() {
	doSomething()
}
`)

		Run([]string{"todo"}, dir)

		output, code := Run([]string{"link", "abc123:validate_input"}, dir)
		if code != 0 {
			t.Fatalf("link failed, exit code = %d, output: %s", code, output)
		}

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0, output: %s", code, output)
		}
		if output != "No changes detected." {
			t.Fatalf("output = %q, want %q", output, "No changes detected.")
		}
	})

	t.Run("link_then_modify_code_then_todo_shows_drift", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="validate_input">input must be validated</spec></specs>`)
		writeCodeFile(t, dir, "main.go", `// #F abc123
func handleRequest() {
	doSomething()
}
`)

		Run([]string{"todo"}, dir)
		Run([]string{"link", "abc123:validate_input"}, dir)
		Run([]string{"todo"}, dir)

		writeCodeFile(t, dir, "main.go", `// #F abc123
func handleRequest() {
	doSomethingElse()
}
`)

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0, output: %s", code, output)
		}
		if !strings.Contains(output, "1 marker has unchecked changes") {
			t.Fatalf("output should mention 1 marker with unchecked changes, got: %s", output)
		}
		if !strings.Contains(output, "abc123") {
			t.Fatalf("output should contain marker id abc123, got: %s", output)
		}
		if !strings.Contains(output, "validate_input") {
			t.Fatalf("output should contain spec id validate_input, got: %s", output)
		}
		if !strings.Contains(output, "drift reset abc123:validate_input") {
			t.Fatalf("output should contain reset command, got: %s", output)
		}
	})

	t.Run("drift_then_reset_clears_drift", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="validate_input">input must be validated</spec></specs>`)
		writeCodeFile(t, dir, "main.go", `// #F abc123
func handleRequest() {
	doSomething()
}
`)

		Run([]string{"todo"}, dir)
		Run([]string{"link", "abc123:validate_input"}, dir)
		Run([]string{"todo"}, dir)

		writeCodeFile(t, dir, "main.go", `// #F abc123
func handleRequest() {
	doSomethingElse()
}
`)

		Run([]string{"todo"}, dir)

		_, code := Run([]string{"reset", "abc123:validate_input"}, dir)
		if code != 0 {
			t.Fatalf("reset failed with non-zero exit code")
		}

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0, output: %s", code, output)
		}
		if output != "No changes detected." {
			t.Fatalf("output = %q, want %q", output, "No changes detected.")
		}
	})
}

func TestCLILinkErrors(t *testing.T) {
	t.Run("link_nonexistent_marker", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)
		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="s1">spec</spec></specs>`)

		output, code := Run([]string{"link", "nonexistent:s1"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
	})

	t.Run("link_nonexistent_spec", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)
		writeCodeFile(t, dir, "main.go", `// #F m1
func a() {}
`)

		output, code := Run([]string{"link", "m1:nonexistent"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
	})

	t.Run("link_duplicate", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)
		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="s1">spec</spec></specs>`)
		writeCodeFile(t, dir, "main.go", `// #F m1
func a() {}
`)

		Run([]string{"todo"}, dir)
		Run([]string{"link", "m1:s1"}, dir)

		output, code := Run([]string{"link", "m1:s1"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code for duplicate link, got 0, output: %s", output)
		}
	})

	t.Run("link_bad_format", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		output, code := Run([]string{"link", "no_colon"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
	})

	t.Run("link_without_init", func(t *testing.T) {
		dir := t.TempDir()
		output, code := Run([]string{"link", "m1:s1"}, dir)
		if code == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", output)
		}
	})
}

func assertTodoCountInOutput(t *testing.T, output string, want int) {
	t.Helper()
	if want == 0 {
		if output != "No changes detected." {
			t.Fatalf("output = %q, want %q", output, "No changes detected.")
		}
		return
	}
	if !strings.Contains(output, fmt.Sprintf("%d. [TODO]", want)) {
		t.Fatalf("output should contain %d todo items, got: %s", want, output)
	}
	lines := strings.Count(output, "[TODO]")
	if lines != want {
		t.Fatalf("output contains %d [TODO] entries, want %d, output: %s", lines, want, output)
	}
}

func assertPinResolutionCount(t *testing.T, dir string, want int) {
	t.Helper()
	store := NewFilePinStore(dir)
	state, err := store.Load()
	assertNoError(t, err)
	if len(state.ResolutionState) != want {
		t.Fatalf("resolution state count = %d, want %d", len(state.ResolutionState), want)
	}
}

func TestCLIManyToManyOneSpecManyMarkers(t *testing.T) {
	t.Run("1_spec_2_markers_full_cycle", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="auth_token_expiry">token must expire</spec></specs>`)
		writeCodeFile(t, dir, "middleware.go", `// #F m1
func authMiddleware() {
	checkExpiry()
}
`)
		writeCodeFile(t, dir, "login.go", `// #F m2
func loginHandler() {
	checkExpiry()
}
`)

		Run([]string{"todo"}, dir)
		Run([]string{"link", "m1:auth_token_expiry"}, dir)
		Run([]string{"link", "m2:auth_token_expiry"}, dir)

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs><spec id="auth_token_expiry">token must expire within 24 hours</spec></specs>`)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 2)

		_, code = Run([]string{"reset", "m1:auth_token_expiry"}, dir)
		if code != 0 {
			t.Fatalf("reset m1 failed")
		}
		assertPinResolutionCount(t, dir, 1)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 1)

		_, code = Run([]string{"reset", "m2:auth_token_expiry"}, dir)
		if code != 0 {
			t.Fatalf("reset m2 failed")
		}
		assertPinResolutionCount(t, dir, 0)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)
	})
}

func TestCLIManyToManyOneMarkerManySpecs(t *testing.T) {
	t.Run("2_specs_1_marker_full_cycle", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs>
			<spec id="validate_file_size">file size must be validated</spec>
			<spec id="scan_for_malware">files must be scanned for malware</spec>
		</specs>`)
		writeCodeFile(t, dir, "upload.go", `// #F m1
func uploadHandler() {
	validateAndScan()
}
`)

		Run([]string{"todo"}, dir)
		Run([]string{"link", "m1:validate_file_size"}, dir)
		Run([]string{"link", "m1:scan_for_malware"}, dir)

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)

		writeCodeFile(t, dir, "upload.go", `// #F m1
func uploadHandler() {
	// forgot to validate!
	upload()
}
`)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 2)

		_, code = Run([]string{"reset", "m1:validate_file_size"}, dir)
		if code != 0 {
			t.Fatalf("reset validate_file_size failed")
		}
		assertPinResolutionCount(t, dir, 1)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 1)

		_, code = Run([]string{"reset", "m1:scan_for_malware"}, dir)
		if code != 0 {
			t.Fatalf("reset scan_for_malware failed")
		}
		assertPinResolutionCount(t, dir, 0)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)
	})
}

func TestCLIManyToManyTwoByTwo(t *testing.T) {
	t.Run("2_specs_2_markers_4_edges_full_cycle", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs>
			<spec id="rate_limit_per_user">per-user rate limiting required</spec>
			<spec id="log_rate_limit_hits">rate limit hits must be logged</spec>
		</specs>`)
		writeCodeFile(t, dir, "middleware.go", `// #F m1
func rateLimitMiddleware() {
	limit()
}
`)
		writeCodeFile(t, dir, "handler.go", `// #F m2
func requestHandler() {
	handle()
}
`)

		Run([]string{"todo"}, dir)
		Run([]string{"link", "m1:rate_limit_per_user"}, dir)
		Run([]string{"link", "m1:log_rate_limit_hits"}, dir)
		Run([]string{"link", "m2:rate_limit_per_user"}, dir)
		Run([]string{"link", "m2:log_rate_limit_hits"}, dir)

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs>
			<spec id="rate_limit_per_user">per-user rate limiting required with 100 req/min</spec>
			<spec id="log_rate_limit_hits">rate limit hits must be logged to syslog</spec>
		</specs>`)
		writeCodeFile(t, dir, "middleware.go", `// #F m1
func rateLimitMiddleware() {
	limitV2()
}
`)
		writeCodeFile(t, dir, "handler.go", `// #F m2
func requestHandler() {
	handleV2()
}
`)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 4)

		_, code = Run([]string{"reset", "m1:rate_limit_per_user"}, dir)
		if code != 0 {
			t.Fatalf("reset 1 failed")
		}
		assertPinResolutionCount(t, dir, 1)
		output, _ = Run([]string{"todo"}, dir)
		assertTodoCountInOutput(t, output, 3)

		_, code = Run([]string{"reset", "m1:log_rate_limit_hits"}, dir)
		if code != 0 {
			t.Fatalf("reset 2 failed")
		}
		assertPinResolutionCount(t, dir, 2)
		output, _ = Run([]string{"todo"}, dir)
		assertTodoCountInOutput(t, output, 2)

		_, code = Run([]string{"reset", "m2:rate_limit_per_user"}, dir)
		if code != 0 {
			t.Fatalf("reset 3 failed")
		}
		assertPinResolutionCount(t, dir, 2)
		output, _ = Run([]string{"todo"}, dir)
		assertTodoCountInOutput(t, output, 1)

		_, code = Run([]string{"reset", "m2:log_rate_limit_hits"}, dir)
		if code != 0 {
			t.Fatalf("reset 4 failed")
		}
		assertPinResolutionCount(t, dir, 0)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)
	})
}

func TestCLIManyToManyThreeByThree(t *testing.T) {
	t.Run("3_specs_3_markers_9_edges_partial_then_full_collapse", func(t *testing.T) {
		dir := t.TempDir()
		Run([]string{"init"}, dir)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs>
			<spec id="validate_amount">amount must be validated</spec>
			<spec id="check_fraud_rules">fraud rules must be checked</spec>
			<spec id="log_transaction">transactions must be logged</spec>
		</specs>`)
		writeCodeFile(t, dir, "card.go", `// #F m1
func cardHandler() {
	processCard()
}
`)
		writeCodeFile(t, dir, "bank.go", `// #F m2
func bankTransferHandler() {
	processBank()
}
`)
		writeCodeFile(t, dir, "wallet.go", `// #F m3
func walletHandler() {
	processWallet()
}
`)

		Run([]string{"todo"}, dir)

		links := []string{
			"m1:validate_amount", "m1:check_fraud_rules", "m1:log_transaction",
			"m2:validate_amount", "m2:check_fraud_rules", "m2:log_transaction",
			"m3:validate_amount", "m3:check_fraud_rules", "m3:log_transaction",
		}
		for _, link := range links {
			_, code := Run([]string{"link", link}, dir)
			if code != 0 {
				t.Fatalf("link %s failed", link)
			}
		}

		output, code := Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)

		writeSpecFile(t, dir, "specs.pin.xml", `<specs>
			<spec id="validate_amount">amount must be validated and positive</spec>
			<spec id="check_fraud_rules">fraud rules must be checked with ML model</spec>
			<spec id="log_transaction">transactions must be logged with audit trail</spec>
		</specs>`)
		writeCodeFile(t, dir, "card.go", `// #F m1
func cardHandler() {
	processCardV2()
}
`)
		writeCodeFile(t, dir, "bank.go", `// #F m2
func bankTransferHandler() {
	processBankV2()
}
`)
		writeCodeFile(t, dir, "wallet.go", `// #F m3
func walletHandler() {
	processWalletV2()
}
`)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 9)

		_, code = Run([]string{"reset", "m1:validate_amount"}, dir)
		if code != 0 {
			t.Fatalf("reset m1:validate_amount failed")
		}
		_, code = Run([]string{"reset", "m2:check_fraud_rules"}, dir)
		if code != 0 {
			t.Fatalf("reset m2:check_fraud_rules failed")
		}
		assertPinResolutionCount(t, dir, 2)

		output, _ = Run([]string{"todo"}, dir)
		assertTodoCountInOutput(t, output, 7)

		for _, link := range links {
			if link == "m1:validate_amount" || link == "m2:check_fraud_rules" {
				continue
			}
			_, code := Run([]string{"reset", link}, dir)
			if code != 0 {
				t.Fatalf("reset %s failed", link)
			}
		}
		assertPinResolutionCount(t, dir, 0)

		output, code = Run([]string{"todo"}, dir)
		if code != 0 {
			t.Fatalf("exit code = %d, output: %s", code, output)
		}
		assertTodoCountInOutput(t, output, 0)

		store := NewFilePinStore(dir)
		state, err := store.Load()
		assertNoError(t, err)
		if len(state.Links) != 9 {
			t.Fatalf("expected 9 links in pin, got %d", len(state.Links))
		}
		if len(state.ResolutionState) != 0 {
			t.Fatalf("expected 0 resolutions after full collapse, got %d", len(state.ResolutionState))
		}
	})
}
