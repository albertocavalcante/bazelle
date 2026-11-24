package incremental

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func atomicNoTemps(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestJSONStoreSave_RoundTripNoTemp(t *testing.T) {
	dir := t.TempDir()
	st := NewJSONStore(dir)
	for i := 0; i < 2; i++ {
		if err := st.Save(&Index{}); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, stateDir))
	for _, e := range entries {
		if e.Name() != stateFile {
			t.Errorf("leftover: %s", e.Name())
		}
	}
}

func TestAtomicWrite_ExistingFileKeepsMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX modes")
	}
	for _, mode := range []os.FileMode{0o600, 0o666, 0o640} {
		dir := t.TempDir()
		path := filepath.Join(dir, "f")
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		if err := atomicWriteFile(path, []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != mode {
			t.Errorf("mode = %v, want %v preserved", fi.Mode().Perm(), mode)
		}
		if b, _ := os.ReadFile(path); string(b) != "new" {
			t.Errorf("content = %q", b)
		}
		atomicNoTemps(t, dir)
	}
}

// A new file gets what os.WriteFile(..., perm) gives under the current umask.
func TestAtomicWrite_NewFileHonoursUmask(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX modes")
	}
	for _, perm := range []os.FileMode{0o644, 0o755} {
		dir := t.TempDir()
		control := filepath.Join(dir, "control")
		if err := os.WriteFile(control, nil, perm); err != nil {
			t.Fatal(err)
		}
		want, _ := os.Stat(control)
		path := filepath.Join(dir, "f")
		if err := atomicWriteFile(path, []byte("x"), perm); err != nil {
			t.Fatal(err)
		}
		got, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Mode().Perm() != want.Mode().Perm() {
			t.Errorf("mode = %v, want %v", got.Mode().Perm(), want.Mode().Perm())
		}
	}
}

// The target is a non-empty directory so the rename fails; the original
// stays intact and no temp is left behind.
func TestAtomicWrite_FailureLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(path, "keep")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(path, []byte("new"), 0o644); err == nil {
		t.Fatal("expected error")
	}
	if b, err := os.ReadFile(keep); err != nil || string(b) != "x" {
		t.Errorf("original damaged: %q %v", b, err)
	}
	atomicNoTemps(t, dir)
}

func TestAtomicWrite_WriteErrorLeavesOriginalAndNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := atomicWriteFunc(path, 0o644, func(f *os.File) error {
		_, _ = f.WriteString("partial")
		return os.ErrInvalid
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if b, _ := os.ReadFile(path); string(b) != "old" {
		t.Errorf("original damaged: %q", b)
	}
	atomicNoTemps(t, dir)
}

func TestAtomicWrite_ConcurrentWritersDoNotCollide(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := atomicWriteFile(path, []byte("payload"), 0o644); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("write: %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "payload" {
		t.Errorf("content = %q", b)
	}
	atomicNoTemps(t, dir)
}
