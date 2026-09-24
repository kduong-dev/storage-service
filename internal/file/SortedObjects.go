package file

import (
	"cmp"
	"slices"
	"strings"
)

// SortedObjects keeps objects ordered by key, then by ID since several
// uploads can share a key, so key prefixes form contiguous ranges.
type SortedObjects []*Object

func compareObjects(left *Object, right *Object) int {
	return cmp.Or(strings.Compare(left.Key, right.Key), strings.Compare(left.ID, right.ID))
}

func (objects *SortedObjects) Insert(object *Object) {
	index, _ := slices.BinarySearchFunc(*objects, object, compareObjects)
	*objects = slices.Insert(*objects, index, object)
}

// IndexAfter returns the index of the first object ordered after the given one.
func (objects SortedObjects) IndexAfter(object *Object) int {
	index, found := slices.BinarySearchFunc(objects, object, compareObjects)
	if found {
		index++
	}
	return index
}

// IndexOfKeyPrefix returns the index of the first object whose key is not
// ordered before the prefix, which is where any keys with that prefix start.
func (objects SortedObjects) IndexOfKeyPrefix(keyPrefix string) int {
	index, _ := slices.BinarySearchFunc(objects, keyPrefix, func(object *Object, keyPrefix string) int {
		return strings.Compare(object.Key, keyPrefix)
	})
	return index
}

type PageInput struct {
	KeyPrefix string
	// After is the object the page starts after, nil for the first page.
	After *Object
	Limit int
}

// Page returns up to Limit objects with the key prefix, starting after
// input.After, and whether more follow.
func (objects SortedObjects) Page(input PageInput) (page SortedObjects, hasMore bool) {
	start := objects.IndexOfKeyPrefix(input.KeyPrefix)
	if input.After != nil {
		start = max(start, objects.IndexAfter(input.After))
	}
	for _, object := range objects[start:] {
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
