package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/co191194/memo-app/internal/memo"
)

type StorageOperatorImpl struct {
	rename func(oldPath, newPath string) error
}

type Memo = memo.Memo

func (mo *StorageOperatorImpl) LoadMemos(path string) ([]Memo, error) {

	path, err := expandPath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Memo{}, nil
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var elements []json.RawMessage
	if err := decoder.Decode(&elements); err != nil {
		return nil, err
	}
	if elements == nil {
		return nil, fmt.Errorf("memo document must be an array")
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected value after memo array")
		}
		return nil, err
	}

	memos := make([]Memo, 0, len(elements))
	for i, element := range elements {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(element, &fields); err != nil {
			return nil, fmt.Errorf("memo %d: %w", i, err)
		}
		if fields == nil {
			return nil, fmt.Errorf("memo %d must be an object", i)
		}
		for _, key := range []string{"id", "title", "body", "created_at", "updated_at"} {
			value, ok := fields[key]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return nil, fmt.Errorf("memo %d: missing or null %s", i, key)
			}
		}
		var item Memo
		if err := json.Unmarshal(element, &item); err != nil {
			return nil, fmt.Errorf("memo %d: %w", i, err)
		}
		memos = append(memos, item)
	}

	return memos, nil
}

func (mo *StorageOperatorImpl) SaveMemos(path string, memos []Memo) error {

	path, err := expandPath(path)
	if err != nil {
		return err
	}

	dirPath := filepath.Dir(path)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dirPath, ".memos-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(memos); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	rename := os.Rename
	if mo.rename != nil {
		rename = mo.rename
	}
	if err := rename(tmpPath, path); err != nil {
		return err
	}

	return nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		path = filepath.Join(home, path[2:])
	}
	return path, nil
}
