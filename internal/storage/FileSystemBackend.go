package storage

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

var _ Backend = (*FileSystemBackend)(nil)

// FileSystemBackend persists part data and assembled objects under a configurable
// root directory:
//
//	<root>/parts/<uploadID>/<partNumber>
//	<root>/objects/<fileID>
type FileSystemBackend struct {
	root string
}

func NewFileSystemBackend(root string) *FileSystemBackend {
	for _, subdirectory := range []string{"parts", "objects"} {
		err := os.MkdirAll(filepath.Join(root, subdirectory), 0o755)
		if err != nil {
			panic(fmt.Sprintf("storage: create subdir %s: %v", subdirectory, err))
		}
	}
	return &FileSystemBackend{root: root}
}

func (backend *FileSystemBackend) WritePart(uploadID string, partNumber int, r io.Reader) (int64, string, error) {
	directory := filepath.Join(backend.root, "parts", uploadID)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return 0, "", fmt.Errorf("storage: create part dir: %w", err)
	}
	path := filepath.Join(directory, fmt.Sprintf("%d", partNumber))
	f, err := os.Create(path)
	if err != nil {
		return 0, "", fmt.Errorf("storage: create part file: %w", err)
	}
	defer f.Close()

	h := md5.New()
	size, err := io.Copy(io.MultiWriter(f, h), r)
	if err != nil {
		return 0, "", fmt.Errorf("storage: write part: %w", err)
	}
	return size, hex.EncodeToString(h.Sum(nil)), nil
}

func (backend *FileSystemBackend) Assemble(uploadID string, fileID string, partNumbers []int) (int64, string, error) {
	sorted := make([]int, len(partNumbers))
	copy(sorted, partNumbers)
	sort.Ints(sorted)

	outPath := filepath.Join(backend.root, "objects", fileID)
	out, err := os.Create(outPath)
	if err != nil {
		return 0, "", fmt.Errorf("storage: create object file: %w", err)
	}
	defer out.Close()

	h := md5.New()
	var total int64
	for _, pn := range sorted {
		partPath := filepath.Join(backend.root, "parts", uploadID, fmt.Sprintf("%d", pn))
		part, err := os.Open(partPath)
		if err != nil {
			return 0, "", fmt.Errorf("storage: open part %d: %w", pn, err)
		}
		n, copyErr := io.Copy(io.MultiWriter(out, h), part)
		part.Close()
		if copyErr != nil {
			return 0, "", fmt.Errorf("storage: copy part %d: %w", pn, copyErr)
		}
		total += n
	}
	return total, hex.EncodeToString(h.Sum(nil)), nil
}

func (backend *FileSystemBackend) Open(fileID string) (io.ReadSeekCloser, error) {
	path := filepath.Join(backend.root, "objects", fileID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage: file %s not found", fileID)
		}
		return nil, fmt.Errorf("storage: open file: %w", err)
	}
	return f, nil
}

func (backend *FileSystemBackend) DeleteParts(uploadID string) error {
	directory := filepath.Join(backend.root, "parts", uploadID)
	if err := os.RemoveAll(directory); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: delete parts: %w", err)
	}
	return nil
}
