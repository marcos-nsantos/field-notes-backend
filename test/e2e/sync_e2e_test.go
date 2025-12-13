package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_Sync_BasicFlow(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "sync-basic@example.com")

	t.Run("sync new notes from client", func(t *testing.T) {
		syncReq := map[string]any{
			"device_id": "device-001",
			"notes": []map[string]any{
				{
					"client_id":  "client-note-1",
					"title":      "Offline Note 1",
					"content":    "Created while offline",
					"updated_at": time.Now().UTC().Format(time.RFC3339),
				},
				{
					"client_id":  "client-note-2",
					"title":      "Offline Note 2",
					"content":    "Another offline note",
					"updated_at": time.Now().UTC().Format(time.RFC3339),
				},
			},
		}

		resp, err := app.post("/sync", syncReq, authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var syncResp map[string]any
		parseResponse(t, resp, &syncResp)

		assert.NotEmpty(t, syncResp["new_cursor"])
		assert.NotNil(t, syncResp["server_notes"])
		assert.NotNil(t, syncResp["conflicts"])

		// Verify notes were created via API
		resp, err = app.get("/notes", authHeader(token))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp map[string]any
		parseResponse(t, resp, &listResp)
		notes := listResp["notes"].([]any)
		assert.Len(t, notes, 2)
	})
}

func TestE2E_Sync_GetServerChanges(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "sync-server@example.com")

	// Create note via API (simulating server-side creation)
	createReq := map[string]any{
		"title":   "Server Note",
		"content": "Created on server",
	}
	resp, err := app.post("/notes", createReq, authHeader(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Sync with empty client notes
	syncReq := map[string]any{
		"device_id": "device-001",
		"notes":     []map[string]any{},
	}

	resp, err = app.post("/sync", syncReq, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var syncResp map[string]any
	parseResponse(t, resp, &syncResp)

	serverNotes := syncResp["server_notes"].([]any)
	assert.Len(t, serverNotes, 1)

	note := serverNotes[0].(map[string]any)
	assert.Equal(t, "Server Note", note["title"])
}

func TestE2E_Sync_WithCursor(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "sync-cursor@example.com")

	// First sync - create note
	syncReq1 := map[string]any{
		"device_id": "device-001",
		"notes": []map[string]any{
			{
				"client_id":  "note-1",
				"title":      "First Note",
				"content":    "First content",
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	resp, err := app.post("/sync", syncReq1, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var syncResp1 map[string]any
	parseResponse(t, resp, &syncResp1)
	cursor1 := syncResp1["new_cursor"].(string)

	// Wait a bit and create another note via API
	time.Sleep(100 * time.Millisecond)

	createReq := map[string]any{
		"title":   "New Server Note",
		"content": "Created after first sync",
	}
	resp, err = app.post("/notes", createReq, authHeader(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Second sync with cursor - should only get new note
	syncReq2 := map[string]any{
		"device_id":   "device-001",
		"sync_cursor": cursor1,
		"notes":       []map[string]any{},
	}

	resp, err = app.post("/sync", syncReq2, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var syncResp2 map[string]any
	parseResponse(t, resp, &syncResp2)

	serverNotes := syncResp2["server_notes"].([]any)
	// Should get the new note created after cursor
	assert.GreaterOrEqual(t, len(serverNotes), 1)
}

func TestE2E_Sync_ConflictClientWins(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "sync-conflict-client@example.com")

	// Create note via API
	createReq := map[string]any{
		"title":     "Original Title",
		"content":   "Original content",
		"client_id": "conflict-note",
	}
	resp, err := app.post("/notes", createReq, authHeader(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var noteResp map[string]any
	parseResponse(t, resp, &noteResp)
	serverUpdatedAt := noteResp["updated_at"].(string)

	// Parse server time and add 1 hour to make client newer
	serverTime, _ := time.Parse(time.RFC3339, serverUpdatedAt)
	clientTime := serverTime.Add(1 * time.Hour)

	// Sync with newer client version
	syncReq := map[string]any{
		"device_id": "device-001",
		"notes": []map[string]any{
			{
				"client_id":  "conflict-note",
				"title":      "Client Updated Title",
				"content":    "Client updated content",
				"updated_at": clientTime.Format(time.RFC3339),
			},
		},
	}

	resp, err = app.post("/sync", syncReq, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var syncResp map[string]any
	parseResponse(t, resp, &syncResp)

	conflicts := syncResp["conflicts"].([]any)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0].(map[string]any)
	assert.Equal(t, "conflict-note", conflict["client_id"])
	assert.Equal(t, "client_wins", conflict["resolution"])

	// Verify the note was updated with client version
	resp, err = app.get("/notes", authHeader(token))
	require.NoError(t, err)

	var listResp map[string]any
	parseResponse(t, resp, &listResp)
	notes := listResp["notes"].([]any)
	assert.Len(t, notes, 1)

	note := notes[0].(map[string]any)
	assert.Equal(t, "Client Updated Title", note["title"])
}

func TestE2E_Sync_ConflictServerWins(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "sync-conflict-server@example.com")

	// Create note via API
	createReq := map[string]any{
		"title":     "Server Title",
		"content":   "Server content",
		"client_id": "conflict-note-2",
	}
	resp, err := app.post("/notes", createReq, authHeader(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var noteResp map[string]any
	parseResponse(t, resp, &noteResp)
	serverUpdatedAt := noteResp["updated_at"].(string)

	// Parse server time and subtract 1 hour to make client older
	serverTime, _ := time.Parse(time.RFC3339, serverUpdatedAt)
	clientTime := serverTime.Add(-1 * time.Hour)

	// Sync with older client version
	syncReq := map[string]any{
		"device_id": "device-001",
		"notes": []map[string]any{
			{
				"client_id":  "conflict-note-2",
				"title":      "Old Client Title",
				"content":    "Old client content",
				"updated_at": clientTime.Format(time.RFC3339),
			},
		},
	}

	resp, err = app.post("/sync", syncReq, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var syncResp map[string]any
	parseResponse(t, resp, &syncResp)

	conflicts := syncResp["conflicts"].([]any)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0].(map[string]any)
	assert.Equal(t, "conflict-note-2", conflict["client_id"])
	assert.Equal(t, "server_wins", conflict["resolution"])

	// Verify the note still has server version
	resp, err = app.get("/notes", authHeader(token))
	require.NoError(t, err)

	var listResp map[string]any
	parseResponse(t, resp, &listResp)
	notes := listResp["notes"].([]any)
	assert.Len(t, notes, 1)

	note := notes[0].(map[string]any)
	assert.Equal(t, "Server Title", note["title"])
}

func TestE2E_Sync_DeletedNotes(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	token := createUserAndLogin(t, app, "sync-delete@example.com")

	// Create note via sync
	syncReq := map[string]any{
		"device_id": "device-001",
		"notes": []map[string]any{
			{
				"client_id":  "deletable-note",
				"title":      "To Be Deleted",
				"content":    "This note will be deleted",
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	resp, err := app.post("/sync", syncReq, authHeader(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify note exists
	resp, err = app.get("/notes", authHeader(token))
	require.NoError(t, err)

	var listResp map[string]any
	parseResponse(t, resp, &listResp)
	notes := listResp["notes"].([]any)
	assert.Len(t, notes, 1)

	// Sync with deleted flag
	syncReq = map[string]any{
		"device_id": "device-001",
		"notes": []map[string]any{
			{
				"client_id":  "deletable-note",
				"title":      "To Be Deleted",
				"content":    "This note will be deleted",
				"updated_at": time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
				"is_deleted": true,
			},
		},
	}

	resp, err = app.post("/sync", syncReq, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Verify note is no longer in list
	resp, err = app.get("/notes", authHeader(token))
	require.NoError(t, err)

	parseResponse(t, resp, &listResp)
	notes = listResp["notes"].([]any)
	assert.Len(t, notes, 0)
}

func TestE2E_Sync_UnknownDevice(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	// Register user but use different device_id for sync
	registerReq := map[string]string{
		"email":    "sync-unknown@example.com",
		"password": "password123",
		"name":     "Test User",
	}
	resp, err := app.post("/auth/register", registerReq, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Login with device-001
	loginReq := map[string]string{
		"email":     "sync-unknown@example.com",
		"password":  "password123",
		"device_id": "device-001",
		"platform":  "ios",
	}
	resp, err = app.post("/auth/login", loginReq, nil)
	require.NoError(t, err)

	var loginResp map[string]any
	parseResponse(t, resp, &loginResp)
	token := loginResp["access_token"].(string)

	// Try to sync with unknown device
	syncReq := map[string]any{
		"device_id": "unknown-device",
		"notes":     []map[string]any{},
	}

	resp, err = app.post("/sync", syncReq, authHeader(token))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errResp map[string]any
	parseResponse(t, resp, &errResp)
	assert.Equal(t, "DEVICE_NOT_FOUND", errResp["code"])
}

func TestE2E_Sync_MultipleDevices(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	// Register user
	registerReq := map[string]string{
		"email":    "sync-multi@example.com",
		"password": "password123",
		"name":     "Multi Device User",
	}
	resp, err := app.post("/auth/register", registerReq, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Login from device 1
	loginReq1 := map[string]string{
		"email":     "sync-multi@example.com",
		"password":  "password123",
		"device_id": "device-1",
		"platform":  "ios",
	}
	resp, err = app.post("/auth/login", loginReq1, nil)
	require.NoError(t, err)
	var loginResp1 map[string]any
	parseResponse(t, resp, &loginResp1)
	token1 := loginResp1["access_token"].(string)

	// Login from device 2
	loginReq2 := map[string]string{
		"email":     "sync-multi@example.com",
		"password":  "password123",
		"device_id": "device-2",
		"platform":  "android",
	}
	resp, err = app.post("/auth/login", loginReq2, nil)
	require.NoError(t, err)
	var loginResp2 map[string]any
	parseResponse(t, resp, &loginResp2)
	token2 := loginResp2["access_token"].(string)

	// Device 1 creates a note
	syncReq1 := map[string]any{
		"device_id": "device-1",
		"notes": []map[string]any{
			{
				"client_id":  "device1-note",
				"title":      "Note from Device 1",
				"content":    "Created on device 1",
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	resp, err = app.post("/sync", syncReq1, authHeader(token1))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Device 2 syncs and should receive the note
	syncReq2 := map[string]any{
		"device_id": "device-2",
		"notes":     []map[string]any{},
	}
	resp, err = app.post("/sync", syncReq2, authHeader(token2))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var syncResp2 map[string]any
	parseResponse(t, resp, &syncResp2)

	serverNotes := syncResp2["server_notes"].([]any)
	assert.Len(t, serverNotes, 1)

	note := serverNotes[0].(map[string]any)
	assert.Equal(t, "Note from Device 1", note["title"])
}
