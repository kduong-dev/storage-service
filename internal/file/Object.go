package file

type Object struct {
	ID          string `json:"id"`
	Namespace   string `json:"namespace"`
	UploadID    string `json:"upload_id"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"` // hex-encoded MD5 of the full file
	CreatedAt   string `json:"created_at"`
}
