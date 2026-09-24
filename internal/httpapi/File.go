package httpapi

import (
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

// toFile converts the stored object to the public response type. The direct
// conversion fails to compile if the two structs drift apart.
func toFile(object *file.Object) *storageservice.File {
	converted := storageservice.File(*object)
	return &converted
}

func toFiles(objects []*file.Object) []*storageservice.File {
	files := make([]*storageservice.File, len(objects))
	for index, object := range objects {
		files[index] = toFile(object)
	}
	return files
}
