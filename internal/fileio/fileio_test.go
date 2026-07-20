package fileio_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"drift/internal/fileio"
)

// TestBeginCreatesDriftDir: Begin succeeds even when .drift/ does not yet
// exist; it creates the directory.
func TestBeginCreatesDriftDir(t *testing.T) {
	dir := t.TempDir()
	driftDir := filepath.Join(dir, ".drift")
	if _, err := os.Stat(driftDir); err == nil {
		t.Fatalf("precondition: .drift/ should not exist")
	}
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer sess.Close()
	if _, err := os.Stat(driftDir); err != nil {
		t.Fatalf(".drift/ should exist after Begin: %v", err)
	}
	if _, err := os.Stat(filepath.Join(driftDir, "state.lock")); err != nil {
		t.Fatalf("state.lock should exist after Begin: %v", err)
	}
}

// TestBeginBlocksOnHeldLock: a second Begin in a separate goroutine blocks
// until the first Session is Closed.
func TestBeginBlocksOnHeldLock(t *testing.T) {
	dir := t.TempDir()
	sess1, err := fileio.Begin(dir)
	if err != nil {
		t.Fatalf("Begin 1: %v", err)
	}

	began := make(chan struct{})
	go func() {
		_, _ = fileio.Begin(dir) // blocks
		close(began)
	}()

	select {
	case <-began:
		t.Fatalf("second Begin returned before Close")
	case <-time.After(100 * time.Millisecond):
		// expected: still blocked
	}

	sess1.Close()

	select {
	case <-began:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatalf("second Begin did not return after Close")
	}
}

// TestCloseReleasesLockAllowsReacquire: after Close, a fresh Begin succeeds.
func TestCloseReleasesLockAllowsReacquire(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	sess2, err := fileio.Begin(dir)
	if err != nil {
		t.Fatalf("Begin after Close: %v", err)
	}
	sess2.Close()
}

// TestCloseIsIdempotent: calling Close twice does not error.
func TestCloseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// TestReadReturnsIsNotExistForMissingFile: Read of a file that does not
// exist returns an error for which os.IsNotExist reports true.
func TestReadReturnsIsNotExistForMissingFile(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	_, err = sess.Read("does-not-exist.xml")
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.IsNotExist, got %v", err)
	}
}

// TestWriteThenReadRoundTrip: Write followed by Read returns the same bytes.
func TestWriteThenReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	want := []byte("<drift version=\"4\"></drift>\n")
	if err := sess.Write("state.xml", want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := sess.Read("state.xml")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("round-trip mismatch: got %q want %q", got, want)
	}
}

// TestWriteOverwritesExisting: Write on an existing file replaces its content.
func TestWriteOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	if err := sess.Write("state.xml", []byte("old\n")); err != nil {
		t.Fatal(err)
	}
	if err := sess.Write("state.xml", []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	got, err := sess.Read("state.xml")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Fatalf("expected overwrite, got %q", got)
	}
}

// TestWriteRejectsBadNames: names with path separators or "." / ".." are
// rejected, since Session only operates on direct children of .drift/.
func TestWriteRejectsBadNames(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	bad := []string{"", ".", "..", "foo/bar", "/abs"}
	if runtime.GOOS == "windows" {
		// On Windows, backslash is also a path separator.
		bad = append(bad, "foo\\bar")
	}
	// On Unix a literal backslash is a valid filename character, not a separator.
	for _, name := range bad {
		if err := sess.Write(name, []byte("x")); err == nil {
			t.Errorf("Write(%q) expected error, got nil", name)
		}
		if _, err := sess.Read(name); err == nil {
			t.Errorf("Read(%q) expected error, got nil", name)
		}
	}
}

// TestWriteAfterClose: Write on a closed Session errors.
func TestWriteAfterClose(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	sess.Close()
	if err := sess.Write("state.xml", []byte("x")); err == nil {
		t.Fatalf("Write after Close should error")
	}
}

// TestConcurrentBeginSerializes: N goroutines each Begin→increment→Close a
// shared counter. Without mutual exclusion, the final count would be less
// than N due to lost updates.
func TestConcurrentBeginSerializes(t *testing.T) {
	dir := t.TempDir()
	const n = 20
	var counter int32
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess, err := fileio.Begin(dir)
			if err != nil {
				t.Errorf("Begin: %v", err)
				return
			}
			defer sess.Close()
			// Race window: read counter, sleep briefly, write back.
			cur := atomic.LoadInt32(&counter)
			time.Sleep(time.Millisecond)
			atomic.StoreInt32(&counter, cur+1)
		}()
	}
	wg.Wait()
	if counter != n {
		t.Fatalf("counter = %d, want %d (lost updates: locking broken)", counter, n)
	}
}

// TestParallelWriteReadConsistency: 50 sequential Write/Read pairs of two
// distinct payloads; every Read must return one payload or the other in full
// — never a partial write. Verifies atomicity of the temp+rename pattern.
func TestParallelWriteReadConsistency(t *testing.T) {
	dir := t.TempDir()
	sess, err := fileio.Begin(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	v1 := bytes.Repeat([]byte("a"), 4096)
	v2 := bytes.Repeat([]byte("b"), 4096)
	if err := sess.Write("state.xml", v1); err != nil {
		t.Fatal(err)
	}
	const iterations = 50
	for i := 0; i < iterations; i++ {
		want := v1
		if i%2 == 1 {
			want = v2
		}
		if err := sess.Write("state.xml", want); err != nil {
			t.Fatalf("iter %d Write: %v", i, err)
		}
		got, err := sess.Read("state.xml")
		if err != nil {
			t.Fatalf("iter %d Read: %v", i, err)
		}
		if !bytes.Equal(got, v1) && !bytes.Equal(got, v2) {
			t.Fatalf("iter %d: partial write detected: got %d bytes (neither v1=%d nor v2=%d)", i, len(got), len(v1), len(v2))
		}
	}
}

// TestCrossProcessMutualExclusion: a subprocess holds the lock; the parent's
// Begin must block. Re-executes the test binary as the "subprocess" using a
// magic env var. Skipped on Windows and under `go test -race`'s ephemeral
// binary path which can make re-exec unreliable.
func TestCrossProcessMutualExclusion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fd inheritance on windows is unreliable in CI")
	}
	if os.Getenv("FILEIO_HOLD_LOCK") == "1" {
		// Subprocess mode: acquire and hold until stdin is closed.
		sess, err := fileio.Begin(os.Args[len(os.Args)-1])
		if err != nil {
			os.Exit(2)
		}
		buf := make([]byte, 1)
		_, _ = os.Stdin.Read(buf)
		_ = sess.Close()
		os.Exit(0)
	}

	dir := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Skipf("cannot locate test binary: %v", err)
	}
	cmd := exec.Command(self, "-test.run=^TestCrossProcessMutualExclusion$", dir)
	cmd.Env = append(os.Environ(), "FILEIO_HOLD_LOCK=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start subprocess: %v", err)
	}
	// Ensure we always release the subprocess even if the test fails early.
	cleanup := func() {
		_, _ = stdin.Write([]byte("q"))
		_ = cmd.Wait()
	}
	defer cleanup()

	// Wait for subprocess to report it has the lock, or time out.
	acquired := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			if _, err := os.Stat(filepath.Join(dir, ".drift", "state.lock")); err == nil {
				// File exists; give flock a moment more.
				time.Sleep(50 * time.Millisecond)
				close(acquired)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		close(acquired)
	}()
	<-acquired

	// Parent's Begin must block.
	began := make(chan error, 1)
	go func() {
		sess, err := fileio.Begin(dir)
		if err != nil {
			began <- err
			return
		}
		sess.Close()
		began <- nil
	}()
	select {
	case err := <-began:
		t.Fatalf("parent Begin returned while subprocess held lock: %v\nstderr: %s", err, stderr.String())
	case <-time.After(300 * time.Millisecond):
		// expected
	}
	cleanup()
}
