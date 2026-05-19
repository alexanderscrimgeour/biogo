package snapshot

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"os"
	"path/filepath"
)

// Save gob-encodes dto, compresses it with gzip, and writes to path.
// Parent directories are created as needed.
func Save(dto SimulationDTO, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if err := gob.NewEncoder(gz).Encode(dto); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}
