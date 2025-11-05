package document

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Document struct {
	Content        string
	FilePath       string
	FileHash       string
	FileSize       int64
	FileModifiedAt time.Time
	Metadata       map[string]interface{}
}

type Loader struct {
	supportedExtensions []string
}

func NewLoader() *Loader {
	return &Loader{
		supportedExtensions: []string{".md", ".txt", ".go", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h", ".hpp"},
	}
}

// LoadFile loads a single file
func (l *Loader) LoadFile(path string) (*Document, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !l.isSupported(ext) {
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	// Get file info
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate file hash
	hash := sha256.Sum256(content)
	fileHash := hex.EncodeToString(hash[:])

	return &Document{
		Content:        string(content),
		FilePath:       path,
		FileHash:       fileHash,
		FileSize:       fileInfo.Size(),
		FileModifiedAt: fileInfo.ModTime(),
		Metadata: map[string]interface{}{
			"extension": ext,
			"filename":  filepath.Base(path),
		},
	}, nil
}

// LoadDirectory loads all supported files from a directory recursively
func (l *Loader) LoadDirectory(dirPath string) ([]*Document, error) {
	var documents []*Document

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !l.isSupported(ext) {
			return nil
		}

		doc, err := l.LoadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to load %s: %v\n", path, err)
			return nil // Continue with other files
		}

		documents = append(documents, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return documents, nil
}

func (l *Loader) isSupported(ext string) bool {
	for _, supported := range l.supportedExtensions {
		if ext == supported {
			return true
		}
	}
	return false
}
