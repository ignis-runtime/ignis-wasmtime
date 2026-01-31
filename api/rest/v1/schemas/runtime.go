package schemas

import "time"

type RuntimeResponse struct {
	ID          string    `json:"id"`
	RuntimeType string    `json:"runtime_type"`
	Hash        string    `json:"hash"`
	S3FilePath  string    `json:"s3_file_path"` // Using local file path in practice
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}