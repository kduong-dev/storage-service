package upload

type Status string

const (
	StatusInitiated Status = "initiated"
	StatusCompleted Status = "completed"
	StatusAborted   Status = "aborted"
)

type Object struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Status      Status `json:"status"`
	// Parts received so far, indexed by part number (1-based).
	Parts     []Part `json:"parts,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Part struct {
	Number   int    `json:"part_number"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"` // hex-encoded MD5 of the part bytes
}
