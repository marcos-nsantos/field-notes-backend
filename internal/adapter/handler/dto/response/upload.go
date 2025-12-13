package response

import (
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/upload"
)

type UploadResponse struct {
	Photo     PhotoResponse `json:"photo"`
	URL       string        `json:"url"`
	SignedURL string        `json:"signed_url,omitempty"`
}

func UploadResultToResponse(result *upload.UploadResult) UploadResponse {
	return UploadResponse{
		Photo:     PhotoFromEntity(result.Photo),
		URL:       result.URL,
		SignedURL: result.SignedURL,
	}
}
