package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_Auth_RegisterAndLogin(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	t.Run("complete auth flow", func(t *testing.T) {
		// 1. Register a new user
		registerReq := map[string]string{
			"email":    "test@example.com",
			"password": "securePassword123",
			"name":     "Test User",
		}

		resp, err := app.post("/auth/register", registerReq, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var registerResp map[string]any
		parseResponse(t, resp, &registerResp)
		assert.Equal(t, "test@example.com", registerResp["email"])
		assert.Equal(t, "Test User", registerResp["name"])
		assert.NotEmpty(t, registerResp["id"])

		// 2. Login with the registered user
		loginReq := map[string]string{
			"email":       "test@example.com",
			"password":    "securePassword123",
			"device_id":   "device-001",
			"device_name": "Test Device",
			"platform":    "ios",
		}

		resp, err = app.post("/auth/login", loginReq, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var loginResp map[string]any
		parseResponse(t, resp, &loginResp)
		assert.NotEmpty(t, loginResp["access_token"])
		assert.NotEmpty(t, loginResp["refresh_token"])
		assert.NotEmpty(t, loginResp["expires_at"])

		accessToken := loginResp["access_token"].(string)
		refreshToken := loginResp["refresh_token"].(string)

		// 3. Access protected endpoint with token
		resp, err = app.get("/notes", authHeader(accessToken))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// 4. Refresh token
		refreshReq := map[string]string{
			"refresh_token": refreshToken,
		}

		resp, err = app.post("/auth/refresh", refreshReq, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var refreshResp map[string]any
		parseResponse(t, resp, &refreshResp)
		newAccessToken := refreshResp["access_token"].(string)
		assert.NotEmpty(t, newAccessToken)
		// Note: tokens may be identical if generated in same second (same claims, same timestamp)

		// 5. Logout
		resp, err = app.post("/auth/logout", nil, authHeader(newAccessToken))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestE2E_Auth_Register_DuplicateEmail(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	registerReq := map[string]string{
		"email":    "duplicate@example.com",
		"password": "password123",
		"name":     "User One",
	}

	// First registration
	resp, err := app.post("/auth/register", registerReq, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Second registration with same email
	registerReq["name"] = "User Two"
	resp, err = app.post("/auth/register", registerReq, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	var errResp map[string]any
	parseResponse(t, resp, &errResp)
	assert.Equal(t, "USER_EXISTS", errResp["code"])
}

func TestE2E_Auth_Login_InvalidCredentials(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	// Register a user
	registerReq := map[string]string{
		"email":    "valid@example.com",
		"password": "correctPassword",
		"name":     "Valid User",
	}
	resp, err := app.post("/auth/register", registerReq, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Try to login with wrong password
	loginReq := map[string]string{
		"email":     "valid@example.com",
		"password":  "wrongPassword",
		"device_id": "device-001",
		"platform":  "ios",
	}

	resp, err = app.post("/auth/login", loginReq, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var errResp map[string]any
	parseResponse(t, resp, &errResp)
	assert.Equal(t, "INVALID_CREDENTIALS", errResp["code"])
}

func TestE2E_Auth_Login_NonExistentUser(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	loginReq := map[string]string{
		"email":     "nonexistent@example.com",
		"password":  "anyPassword",
		"device_id": "device-001",
		"platform":  "ios",
	}

	resp, err := app.post("/auth/login", loginReq, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_Auth_ProtectedEndpoint_NoToken(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	resp, err := app.get("/notes", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_Auth_ProtectedEndpoint_InvalidToken(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	resp, err := app.get("/notes", authHeader("invalid-token"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_Auth_Register_ValidationErrors(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	tests := []struct {
		name    string
		request map[string]string
	}{
		{
			name: "missing email",
			request: map[string]string{
				"password": "password123",
				"name":     "Test User",
			},
		},
		{
			name: "invalid email",
			request: map[string]string{
				"email":    "not-an-email",
				"password": "password123",
				"name":     "Test User",
			},
		},
		{
			name: "short password",
			request: map[string]string{
				"email":    "test@example.com",
				"password": "short",
				"name":     "Test User",
			},
		},
		{
			name: "missing name",
			request: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := app.post("/auth/register", tc.request, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			resp.Body.Close()
		})
	}
}

func TestE2E_Auth_MultipleDevices(t *testing.T) {
	app := setupTestApp(t)
	defer app.cleanup(t)

	// Register a user
	registerReq := map[string]string{
		"email":    "multidevice@example.com",
		"password": "password123",
		"name":     "Multi Device User",
	}
	resp, err := app.post("/auth/register", registerReq, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Login from device 1
	loginReq1 := map[string]string{
		"email":       "multidevice@example.com",
		"password":    "password123",
		"device_id":   "iphone-001",
		"device_name": "iPhone",
		"platform":    "ios",
	}
	resp, err = app.post("/auth/login", loginReq1, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp1 map[string]any
	parseResponse(t, resp, &loginResp1)
	token1 := loginResp1["access_token"].(string)

	// Login from device 2
	loginReq2 := map[string]string{
		"email":       "multidevice@example.com",
		"password":    "password123",
		"device_id":   "android-001",
		"device_name": "Android Phone",
		"platform":    "android",
	}
	resp, err = app.post("/auth/login", loginReq2, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp2 map[string]any
	parseResponse(t, resp, &loginResp2)
	token2 := loginResp2["access_token"].(string)

	// Both tokens should work (note: tokens may be equal if generated in same second
	// since JWT doesn't include device_id in claims)
	resp, err = app.get("/notes", authHeader(token1))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = app.get("/notes", authHeader(token2))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
