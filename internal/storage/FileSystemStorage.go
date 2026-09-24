package storage

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/kduong-dev/goutil/fatal"
)

var _ Storage = (*FileSystemStorage)(nil)

type FileSystemStorage struct {
	root string
}

type NewFileSystemStorageInput struct {
	Root string
}

func NewFileSystemStorage(input NewFileSystemStorageInput) *FileSystemStorage {
	subDirectories := []string{"parts", "objects"}
	for _, subdirectory := range subDirectories {
		path := filepath.Join(input.Root, subdirectory)
		err := os.MkdirAll(path, 0o755)
		fatal.OnError(err)
	}
	return &FileSystemStorage{
		root: input.Root,
	}
}

func (storage *FileSystemStorage) getSortedPartNumbers(partNumbers []int) []int {
	sorted := make([]int, len(partNumbers))
	copy(sorted, partNumbers)
	sort.Ints(sorted)
	return sorted
}

func (storage *FileSystemStorage) getPartsDirectory(uploadID string) string {
	return filepath.Join(storage.root, "parts", uploadID)
}

func (storage *FileSystemStorage) InitialiseUpload(ctx context.Context, uploadID string) error {
	partsDirectory := storage.getPartsDirectory(uploadID)
	err := os.MkdirAll(partsDirectory, 0o755)
	fatal.OnError(err)
	return nil
}

func (storage *FileSystemStorage) UploadPart(ctx context.Context, input UploadPartInput) (output *UploadPartOutput, err error) {
	partsDirectory := storage.getPartsDirectory(input.UploadID)
	partPath := filepath.Join(partsDirectory, strconv.Itoa(input.PartNumber))
	file, err := os.Create(partPath)
	fatal.OnError(err)
	defer file.Close()
	hash := md5.New()
	writer := io.MultiWriter(file, hash)
	size, err := io.Copy(writer, input.Reader)
	fatal.OnError(err)
	output = &UploadPartOutput{
		Size:     size,
		Checksum: hex.EncodeToString(hash.Sum(nil)),
	}
	return
}

func (storage *FileSystemStorage) CompleteUpload(ctx context.Context, input CompleteUploadInput) (output *CompleteUploadOutput, err error) {
	sortedPartNumbers := storage.getSortedPartNumbers(input.PartNumbers)
	path := filepath.Join(storage.root, "objects", input.Key)
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	fatal.OnError(err)
	file, err := os.Create(path)
	fatal.OnError(err)
	defer file.Close()
	partsDirectory := storage.getPartsDirectory(input.UploadID)
	hash := md5.New()
	var total int64
	for _, partNumber := range sortedPartNumbers {
		partPath := filepath.Join(partsDirectory, strconv.Itoa(partNumber))
		part, err := os.Open(partPath)
		fatal.OnError(err)
		writer := io.MultiWriter(file, hash)
		n, err := io.Copy(writer, part)
		part.Close()
		fatal.OnError(err)
		total += n
	}
	err = os.RemoveAll(partsDirectory)
	fatal.OnError(err)
	output = &CompleteUploadOutput{
		Size:     total,
		Checksum: hex.EncodeToString(hash.Sum(nil)),
	}
	return
}

func (storage *FileSystemStorage) AbortUpload(ctx context.Context, uploadID string) error {
	err := os.RemoveAll(storage.getPartsDirectory(uploadID))
	fatal.OnError(err)
	return nil
}

func (storage *FileSystemStorage) OpenFile(key string) (io.ReadSeekCloser, error) {
	path := filepath.Join(storage.root, "objects", key)
	return os.Open(path)
}
