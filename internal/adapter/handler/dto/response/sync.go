package response

import (
	"time"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/sync"
)

type SyncResponse struct {
	ServerNotes []NoteResponse     `json:"server_notes"`
	NewCursor   time.Time          `json:"new_cursor"`
	Conflicts   []ConflictResponse `json:"conflicts"`
}

type ConflictResponse struct {
	ClientID      string        `json:"client_id"`
	Resolution    string        `json:"resolution"`
	ServerVersion *NoteResponse `json:"server_version,omitempty"`
}

func SyncResultToResponse(result *sync.SyncResult) SyncResponse {
	resp := SyncResponse{
		ServerNotes: make([]NoteResponse, 0, len(result.ServerNotes)),
		NewCursor:   result.NewCursor,
		Conflicts:   make([]ConflictResponse, 0, len(result.Conflicts)),
	}

	for _, n := range result.ServerNotes {
		resp.ServerNotes = append(resp.ServerNotes, NoteFromEntity(&n))
	}

	for _, c := range result.Conflicts {
		conflict := ConflictResponse{
			ClientID:   c.ClientID,
			Resolution: c.Resolution,
		}
		if c.ServerVersion != nil {
			serverNote := NoteFromEntity(c.ServerVersion)
			conflict.ServerVersion = &serverNote
		}
		resp.Conflicts = append(resp.Conflicts, conflict)
	}

	return resp
}

func SyncNotesFromEntities(notes []entity.Note) []NoteResponse {
	result := make([]NoteResponse, 0, len(notes))
	for _, n := range notes {
		result = append(result, NoteFromEntity(&n))
	}
	return result
}
