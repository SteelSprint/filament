package driftpin

import (
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
