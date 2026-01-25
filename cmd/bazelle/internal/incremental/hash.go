package incremental

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// HashFile computes xxHash64 of file contents, returns hex string.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := xxhash.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashBytes computes xxHash64 of bytes, returns hex string.
func HashBytes(data []byte) string {
	h := xxhash.Sum64(data)
	var buf [8]byte
	buf[0] = byte(h >> 56)
	buf[1] = byte(h >> 48)
	buf[2] = byte(h >> 40)
	buf[3] = byte(h >> 32)
	buf[4] = byte(h >> 24)
	buf[5] = byte(h >> 16)
	buf[6] = byte(h >> 8)
	buf[7] = byte(h)
	return hex.EncodeToString(buf[:])
}
