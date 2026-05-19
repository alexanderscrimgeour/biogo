package snapshot

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Load decompresses and gob-decodes a save file from path.
func Load(path string) (SimulationDTO, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SimulationDTO{}, err
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return SimulationDTO{}, err
	}
	defer gz.Close()
	var dto SimulationDTO
	if err := gob.NewDecoder(gz).Decode(&dto); err != nil {
		return SimulationDTO{}, err
	}
	return dto, nil
}

// SaveInfo describes one save file on disk.
type SaveInfo struct {
	Name    string
	Path    string
	ModTime time.Time
}

// ListSaves returns all .biogosave files in dir, sorted newest-first.
// Returns (nil, nil) when the directory does not exist.
func ListSaves(dir string) ([]SaveInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var saves []SaveInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".biogosave" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".biogosave")
		saves = append(saves, SaveInfo{
			Name:    name,
			Path:    filepath.Join(dir, e.Name()),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(saves, func(i, j int) bool {
		return saves[i].ModTime.After(saves[j].ModTime)
	})
	return saves, nil
}
