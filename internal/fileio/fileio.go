package fileio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Session is the sole gateway for reads and writes to files under .drift/.
// Construction via Begin acquires an exclusive advisory lock on
// .drift/state.lock; the lock is held until Close. All Read/Write calls
// during a transaction must route through the same Session. Close releases
// the lock and is safe to call multiple times.
type Session struct {
	dir      string // project root (parent of .drift/)
	driftDir string // absolute path to .drift/
	lockFile *os.File
	mu       sync.Mutex
	closed   bool
}

// driftSubdir is the name of the drift state directory inside a project.
const driftSubdir = ".drift"

// lockFileName is the name of the lock file inside .drift/.
const lockFileName = "state.lock"

// D! id=fbeg range-start

// Begin acquires an exclusive advisory lock on .drift/state.lock (creating it
// if needed) and returns a Session through which all .drift/ I/O during the
// transaction must route. Blocks until the lock is acquired. The lock is
// auto-released when the Session is Closed or the process exits.
//
// Concurrent Begin calls from different processes serialize. Within the same
// process, concurrent Begin calls block on each other (flock is not reentrant
// across separate file descriptors of the same file).
func Begin(dir string) (*Session, error) {
	driftDir := filepath.Join(dir, driftSubdir)
	if err := os.MkdirAll(driftDir, 0755); err != nil {
		return nil, fmt.Errorf("fileio: create %s: %w", driftDir, err)
	}
	lockPath := filepath.Join(driftDir, lockFileName)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("fileio: open lock %s: %w", lockPath, err)
	}
	if err := lockFileExclusive(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("fileio: acquire lock %s: %w", lockPath, err)
	}
	return &Session{
		dir:      dir,
		driftDir: driftDir,
		lockFile: f,
	}, nil
}

// Close releases the lock and closes the lock file descriptor. It is safe to
// call multiple times; subsequent calls are no-ops.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.lockFile == nil {
		return nil
	}
	return unlockFile(s.lockFile)
}

// D! id=fbeg range-end

// D! id=fread range-start

// Read returns the bytes of .drift/<name>. A missing file returns an error
// for which os.IsNotExist reports true. Read relies on the lock held since
// Begin; it does not take the lock per-call.
func (s *Session) Read(name string) ([]byte, error) {
	if err := s.checkOpen(); err != nil {
		return nil, err
	}
	if err := validName(name); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(s.driftDir, name))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// D! id=fread range-end

// D! id=fwrite range-start

// Write atomically writes .drift/<name> by writing to a temp file in the same
// directory (so os.Rename never crosses a filesystem boundary) and then
// renaming it over the target. A crash at any point leaves either the old
// content or the new content at the target — never a partial write. Write
// relies on the lock held since Begin; it does not take the lock per-call.
func (s *Session) Write(name string, data []byte) error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	if err := validName(name); err != nil {
		return err
	}
	target := filepath.Join(s.driftDir, name)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("fileio: write temp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, target); err != nil {
		// Best-effort cleanup of orphan temp file; caller may still see it.
		_ = os.Remove(tmp)
		return fmt.Errorf("fileio: rename %s -> %s: %w", tmp, target, err)
	}
	return nil
}

// D! id=fwrite range-end

// checkOpen returns an error if the Session has been Closed.
func (s *Session) checkOpen() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("fileio: Session already Closed")
	}
	return nil
}

// validName rejects empty names, absolute paths, and names containing a path
// separator — Session only reads/writes direct children of .drift/.
func validName(name string) error {
	if name == "" {
		return errors.New("fileio: empty name")
	}
	if name == "." || name == ".." {
		return errors.New("fileio: invalid name " + name)
	}
	for i := 0; i < len(name); i++ {
		if name[i] == os.PathSeparator || name[i] == '/' {
			return errors.New("fileio: name must not contain a path separator: " + name)
		}
	}
	return nil
}
