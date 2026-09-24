package upload

type Status string

const (
	StatusInitiated Status = "initiated"
	StatusCompleted Status = "completed"
	StatusAborted   Status = "aborted"
)
