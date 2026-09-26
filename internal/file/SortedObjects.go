package file

import (
	"cmp"
	"slices"
	"strings"

	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

// SortedObjects orders objects by key, then by ID since several files can
// share a key, so key prefixes form contiguous ranges. Adds are appended and
// sorted on the next read, so replaying a log costs one sort rather than a
// shifting insert per object.
type SortedObjects struct {
	objects  []*storageservice.FileObject
	unsorted bool
}

func compareObjects(left *storageservice.FileObject, right *storageservice.FileObject) int {
	return cmp.Or(strings.Compare(left.Key, right.Key), strings.Compare(left.ID, right.ID))
}

func (sortedObjects *SortedObjects) Add(object *storageservice.FileObject) {
	if length := len(sortedObjects.objects); length > 0 && compareObjects(sortedObjects.objects[length-1], object) > 0 {
		sortedObjects.unsorted = true
	}
	sortedObjects.objects = append(sortedObjects.objects, object)
}

func (sortedObjects *SortedObjects) sort() {
	if sortedObjects.unsorted {
		slices.SortFunc(sortedObjects.objects, compareObjects)
		sortedObjects.unsorted = false
	}
}

// Remove deletes the object, matched by key and ID, if present.
func (sortedObjects *SortedObjects) Remove(object *storageservice.FileObject) {
	sortedObjects.sort()
	if index, found := slices.BinarySearchFunc(sortedObjects.objects, object, compareObjects); found {
		sortedObjects.objects = slices.Delete(sortedObjects.objects, index, index+1)
	}
}

// IndexAfter returns the index of the first object ordered after the given one.
func (sortedObjects *SortedObjects) IndexAfter(object *storageservice.FileObject) int {
	sortedObjects.sort()
	index, found := slices.BinarySearchFunc(sortedObjects.objects, object, compareObjects)
	if found {
		index++
	}
	return index
}

// IndexOfKeyPrefix returns the index of the first object whose key is not
// ordered before the prefix, which is where any keys with that prefix start.
func (sortedObjects *SortedObjects) IndexOfKeyPrefix(keyPrefix string) int {
	sortedObjects.sort()
	index, _ := slices.BinarySearchFunc(sortedObjects.objects, keyPrefix, func(object *storageservice.FileObject, keyPrefix string) int {
		return strings.Compare(object.Key, keyPrefix)
	})
	return index
}

type PageInput struct {
	KeyPrefix string
	// After is the object the page starts after, nil for the first page.
	After *storageservice.FileObject
	Limit int
}

// Page returns up to Limit objects with the key prefix, starting after
// input.After, and whether more follow.
func (sortedObjects *SortedObjects) Page(input PageInput) (page []*storageservice.FileObject, hasMore bool) {
	start := sortedObjects.IndexOfKeyPrefix(input.KeyPrefix)
	if input.After != nil {
		start = max(start, sortedObjects.IndexAfter(input.After))
	}
	for _, object := range sortedObjects.objects[start:] {
		if !strings.HasPrefix(object.Key, input.KeyPrefix) {
			break
		}
		if len(page) == input.Limit {
			return page, true
		}
		page = append(page, object)
	}
	return page, false
}
