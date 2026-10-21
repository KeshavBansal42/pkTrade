// Package testutil provides helpers shared by the pkTrade test suites.
package testutil

import (
	"compress/gzip"
	"io"
	"os"
	"testing"
)

// ReadGzip reads a gzip-compressed fixture from disk.
func ReadGzip(t *testing.T, path string) []byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", path, err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("gzip fixture %s: %v", path, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}
