package e2e_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a user and get access token
func createUserAndLogin(t *testing.T, app *TestApp, email string) string {
	t.Helper()

	registerReq := map[string]string{
		"email":    email,
		"password": "password123",
		"name":     "Test User",
	}
	resp, err := app.post("/auth/register", registerReq, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	loginReq := map[string]string{
		"email":     email,
		"password":  "password123",
		"device_id": "device-001",
		"platform":  "ios",
	}
	resp, err = app.post("/auth/login", loginReq, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]any
	parseResponse(t, resp, &loginResp)
	return loginResp["access_token"].(string)
}

func TestE2E_Notes_CRUD(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "notes-crud@example.com")

	var noteID string

	t.Run("create note", func(t *testing.T) {
		createReq := map[string]any{
			"title":   "My First Note",
			"content": "This is the content of my first note.",
		}

		resp, err := app.post("/notes", createReq, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var noteResp map[string]any
		parseResponse(t, resp, &noteResp)

		noteID = noteResp["id"].(string)
		assert.NotEmpty(t, noteID)
		assert.Equal(t, "My First Note", noteResp["title"])
		assert.Equal(t, "This is the content of my first note.", noteResp["content"])
		assert.NotEmpty(t, noteResp["created_at"])
		assert.NotEmpty(t, noteResp["updated_at"])
	})

	t.Run("get note by id", func(t *testing.T) {
		resp, err := app.get("/notes/"+noteID, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var noteResp map[string]any
		parseResponse(t, resp, &noteResp)

		assert.Equal(t, noteID, noteResp["id"])
		assert.Equal(t, "My First Note", noteResp["title"])
	})

	t.Run("list notes", func(t *testing.T) {
		resp, err := app.get("/notes", authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp map[string]any
		parseResponse(t, resp, &listResp)

		notes := listResp["notes"].([]any)
		assert.Len(t, notes, 1)

		pagination := listResp["pagination"].(map[string]any)
		assert.Equal(t, float64(1), pagination["total_items"])
	})

	t.Run("update note", func(t *testing.T) {
		updateReq := map[string]any{
			"title":   "Updated Title",
			"content": "Updated content here.",
		}

		resp, err := app.put("/notes/"+noteID, updateReq, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var noteResp map[string]any
		parseResponse(t, resp, &noteResp)

		assert.Equal(t, "Updated Title", noteResp["title"])
		assert.Equal(t, "Updated content here.", noteResp["content"])
	})

	t.Run("delete note", func(t *testing.T) {
		resp, err := app.delete("/notes/"+noteID, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()

		// Verify note is deleted
		resp, err = app.get("/notes/"+noteID, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestE2E_Notes_WithLocation(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "notes-location@example.com")

	t.Run("create note with location", func(t *testing.T) {
		createReq := map[string]any{
			"title":     "Field Note",
			"content":   "Observation from the field.",
			"latitude":  40.7128,
			"longitude": -74.0060,
			"altitude":  10.5,
			"accuracy":  5.0,
		}

		resp, err := app.post("/notes", createReq, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var noteResp map[string]any
		parseResponse(t, resp, &noteResp)

		location := noteResp["location"].(map[string]any)
		assert.InDelta(t, 40.7128, location["latitude"], 0.0001)
		assert.InDelta(t, -74.0060, location["longitude"], 0.0001)
		assert.InDelta(t, 10.5, location["altitude"], 0.1)
		assert.InDelta(t, 5.0, location["accuracy"], 0.1)
	})
}

func TestE2E_Notes_Pagination(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "notes-pagination@example.com")

	// Create 25 notes
	for i := 0; i < 25; i++ {
		createReq := map[string]any{
			"title":   fmt.Sprintf("Note %d", i+1),
			"content": fmt.Sprintf("Content for note %d", i+1),
		}
		resp, err := app.post("/notes", createReq, authHeader(token))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	}

	t.Run("first page", func(t *testing.T) {
		resp, err := app.get("/notes?page=1&per_page=10", authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp map[string]any
		parseResponse(t, resp, &listResp)

		notes := listResp["notes"].([]any)
		assert.Len(t, notes, 10)

		pagination := listResp["pagination"].(map[string]any)
		assert.Equal(t, float64(25), pagination["total_items"])
		assert.Equal(t, float64(3), pagination["total_pages"])
		assert.Equal(t, true, pagination["has_next"])
		assert.Equal(t, false, pagination["has_prev"])
	})

	t.Run("second page", func(t *testing.T) {
		resp, err := app.get("/notes?page=2&per_page=10", authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp map[string]any
		parseResponse(t, resp, &listResp)

		notes := listResp["notes"].([]any)
		assert.Len(t, notes, 10)

		pagination := listResp["pagination"].(map[string]any)
		assert.Equal(t, true, pagination["has_next"])
		assert.Equal(t, true, pagination["has_prev"])
	})

	t.Run("last page", func(t *testing.T) {
		resp, err := app.get("/notes?page=3&per_page=10", authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp map[string]any
		parseResponse(t, resp, &listResp)

		notes := listResp["notes"].([]any)
		assert.Len(t, notes, 5)

		pagination := listResp["pagination"].(map[string]any)
		assert.Equal(t, false, pagination["has_next"])
		assert.Equal(t, true, pagination["has_prev"])
	})
}

func TestE2E_Notes_Idempotency(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "notes-idempotent@example.com")

	clientID := "unique-client-id-12345"

	// Create note with client_id
	createReq := map[string]any{
		"title":     "Idempotent Note",
		"content":   "This note should only be created once.",
		"client_id": clientID,
	}

	resp, err := app.post("/notes", createReq, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var firstResp map[string]any
	parseResponse(t, resp, &firstResp)
	firstID := firstResp["id"].(string)

	// Try to create same note again with same client_id
	resp, err = app.post("/notes", createReq, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var secondResp map[string]any
	parseResponse(t, resp, &secondResp)
	secondID := secondResp["id"].(string)

	// Should return the same note (idempotent)
	assert.Equal(t, firstID, secondID)

	// Should still have only 1 note
	resp, err = app.get("/notes", authHeader(token))
	require.NoError(t, err)

	var listResp map[string]any
	parseResponse(t, resp, &listResp)
	notes := listResp["notes"].([]any)
	assert.Len(t, notes, 1)
}

func TestE2E_Notes_UserIsolation(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	// Create two users
	token1 := createUserAndLogin(t, app, "user1@example.com")
	token2 := createUserAndLogin(t, app, "user2@example.com")

	// User 1 creates a note
	createReq := map[string]any{
		"title":   "User 1's Private Note",
		"content": "This belongs to user 1.",
	}
	resp, err := app.post("/notes", createReq, authHeader(token1))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var noteResp map[string]any
	parseResponse(t, resp, &noteResp)
	noteID := noteResp["id"].(string)

	// User 2 should not see user 1's notes
	resp, err = app.get("/notes", authHeader(token2))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp map[string]any
	parseResponse(t, resp, &listResp)
	notes := listResp["notes"].([]any)
	assert.Len(t, notes, 0)

	// User 2 should not be able to access user 1's note directly
	resp, err = app.get("/notes/"+noteID, authHeader(token2))
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// User 2 should not be able to update user 1's note
	updateReq := map[string]any{
		"title": "Hacked Title",
	}
	resp, err = app.put("/notes/"+noteID, updateReq, authHeader(token2))
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// User 2 should not be able to delete user 1's note
	resp, err = app.delete("/notes/"+noteID, authHeader(token2))
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_Notes_ValidationErrors(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "notes-validation@example.com")

	tests := []struct {
		name    string
		request map[string]any
	}{
		{
			name: "missing title",
			request: map[string]any{
				"content": "Content without title",
			},
		},
		{
			name: "missing content",
			request: map[string]any{
				"title": "Title without content",
			},
		},
		{
			name: "invalid latitude",
			request: map[string]any{
				"title":     "Note",
				"content":   "Content",
				"latitude":  999.0,
				"longitude": 0.0,
			},
		},
		{
			name: "invalid longitude",
			request: map[string]any{
				"title":     "Note",
				"content":   "Content",
				"latitude":  0.0,
				"longitude": 999.0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := app.post("/notes", tc.request, authHeader(token))
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			resp.Body.Close()
		})
	}
}

func TestE2E_Notes_NotFound(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "notes-notfound@example.com")

	nonExistentID := "00000000-0000-0000-0000-000000000000"

	t.Run("get non-existent note", func(t *testing.T) {
		resp, err := app.get("/notes/"+nonExistentID, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("update non-existent note", func(t *testing.T) {
		updateReq := map[string]any{
			"title": "Updated Title",
		}
		resp, err := app.put("/notes/"+nonExistentID, updateReq, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("delete non-existent note", func(t *testing.T) {
		resp, err := app.delete("/notes/"+nonExistentID, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})
}
