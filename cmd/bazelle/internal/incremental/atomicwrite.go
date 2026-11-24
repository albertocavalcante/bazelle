package incremental

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// atomicWriteFile replaces path with data; see atomicWriteFunc.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	return atomicWriteFunc(path, perm, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}

// atomicWriteFunc replaces path with whatever write puts in a uniquely
// named temp file in the same directory: write, fsync, close, rename,
// then a best-effort fsync of the directory. The fsyncs matter even for a
// rebuildable cache: without them a power loss can leave an empty or
// truncated file at the final name, which JSONStore.Load reports as an
// error. An existing target keeps its current permission bits (the temp file is created 0600 and set to
// the target's own mode before the rename, so it is never wider than the
// original); a new target gets perm filtered by the umask, as
// os.WriteFile gave it. On any failure the temp file is removed and the
// original is untouched.
func atomicWriteFunc(path string, perm os.FileMode, write func(*os.File) error) error {
	dir := filepath.Dir(path)
	createPerm, keep, preserve := perm, os.FileMode(0), false
	if fi, err := os.Stat(path); err == nil && fi.Mode().IsRegular() {
		createPerm, keep, preserve = 0o600, fi.Mode().Perm(), true
	}
	f, tmp, err := atomicCreateTemp(dir, filepath.Base(path), createPerm)
	if err != nil {
		return err
	}
	err = write(f)
	if err == nil && preserve && runtime.GOOS != "windows" {
		err = f.Chmod(keep)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	atomicSyncDir(dir)
	return nil
}

// atomicCreateTemp creates dir/.<base>.<random>.tmp with O_EXCL,
// retrying on name collisions.
func atomicCreateTemp(dir, base string, perm os.FileMode) (*os.File, string, error) {
	for range 100 {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, "", fmt.Errorf("temp name: %w", err)
		}
		name := filepath.Join(dir, "."+base+"."+hex.EncodeToString(b[:])+".tmp")
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("create temp: %w", err)
		}
		return f, name, nil
	}
	return nil, "", fmt.Errorf("create temp in %s: too many name collisions", dir)
}

// atomicSyncDir fsyncs dir so a rename is durable. Best effort: errors
// (including Windows refusing to open a directory) are ignored.
func atomicSyncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
}
