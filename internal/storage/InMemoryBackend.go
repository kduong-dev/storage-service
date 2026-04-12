package storage

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
)

var _ Backend = (*InMemoryBackend)(nil)

// InMemoryBackend holds all data in process memory. Useful for testing and
// local development; all data is lost on restart.
type InMemoryBackend struct {
	mu sync.RWMutex
	// parts[uploadID][partNumber] = bytes
	parts map[string]map[int][]byte
	// objects[fileID] = assembled bytes
	objects map[string][]byte
}

func NewInMemoryBackend() *InMemoryBackend {
	return &InMemoryBackend{
		parts:   make(map[string]map[int][]byte),
		objects: make(map[string][]byte),
	}
}

func (backend *InMemoryBackend) WritePart(uploadID string, partNumber int, r io.Reader) (int64, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, "", fmt.Errorf("storage: read part body: %w", err)
	}
	sum := md5.Sum(data)
	checksum := hex.EncodeToString(sum[:])

	backend.mu.Lock()
	defer backend.mu.Unlock()
	if _, ok := backend.parts[uploadID]; !ok {
		backend.parts[uploadID] = make(map[int][]byte)
	}
	backend.parts[uploadID][partNumber] = data
	return int64(len(data)), checksum, nil
}

func (backend *InMemoryBackend) Assemble(uploadID string, fileID string, partNumbers []int) (int64, string, error) {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	uploadParts, ok := backend.parts[uploadID]
	if !ok {
		return 0, "", errors.New("storage: no parts found for upload")
	}

	sorted := make([]int, len(partNumbers))
	copy(sorted, partNumbers)
	sort.Ints(sorted)

	h := md5.New()
	var buf bytes.Buffer
	for _, pn := range sorted {
		part, exists := uploadParts[pn]
		if !exists {
			return 0, "", fmt.Errorf("storage: part %d not found", pn)
		}
		buf.Write(part)
		h.Write(part)
	}

	assembled := buf.Bytes()
	backend.objects[fileID] = assembled
	return int64(len(assembled)), hex.EncodeToString(h.Sum(nil)), nil
}

func (backend *InMemoryBackend) Open(fileID string) (io.ReadSeekCloser, error) {
	backend.mu.RLock()
	defer backend.mu.RUnlock()
	data, ok := backend.objects[fileID]
	if !ok {
		return nil, fmt.Errorf("storage: file %s not found", fileID)
	}
	return &nopReadSeekCloser{Reader: bytes.NewReader(data)}, nil
}

// nopReadSeekCloser adds a no-op Close to bytes.Reader so it satisfies
// io.ReadSeekCloser.
type nopReadSeekCloser struct {
	*bytes.Reader
}

func (n *nopReadSeekCloser) Close() error { return nil }

func (backend *InMemoryBackend) DeleteParts(uploadID string) error {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	delete(backend.parts, uploadID)
	return nil
}
