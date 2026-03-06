package smoke

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// todoResponse matches the API JSON shape (Id, Description, Completed).
type todoResponse struct {
	Id          string `json:"Id"`
	Description string `json:"Description"`
	Completed   bool   `json:"Completed"`
}

func getBaseURL(t *testing.T) string {
	base := os.Getenv("TEST_APP_URL")
	if base == "" {
		t.Skip("TEST_APP_URL not set, skipping HTTP smoke test")
	}
	return base
}

func TestSmoke_Healthz(t *testing.T) {
	base := getBaseURL(t)
	resp, err := http.Get(base + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body["alive"].(bool))
	assert.Contains(t, []interface{}{"ready", "connecting"}, body["db"])
}

func TestSmoke_Readyz_DBConnected(t *testing.T) {
	base := getBaseURL(t)
	resp, err := http.Get(base + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "readyz must return 200 when DB is connected")
}

func TestSmoke_CRUD_Sequence(t *testing.T) {
	base := getBaseURL(t)
	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Create
	form := url.Values{}
	form.Set("description", "smoke-test")
	req, err := http.NewRequest(http.MethodPost, base+"/todo", bytes.NewBufferString(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var createResult []todoResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createResult))
	require.Len(t, createResult, 1)
	id := createResult[0].Id
	require.NotEmpty(t, id)
	assert.Equal(t, "smoke-test", createResult[0].Description)
	assert.False(t, createResult[0].Completed)

	// 2. List incomplete — item appears
	resp2, err := http.Get(base + "/todo-incomplete")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	var incomplete []todoResponse
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&incomplete))
	var found bool
	for _, item := range incomplete {
		if item.Id == id {
			found = true
			break
		}
	}
	assert.True(t, found, "created item should appear in todo-incomplete")

	// 3. Update to completed
	form.Set("completed", "true")
	req3, err := http.NewRequest(http.MethodPost, base+"/todo/"+id, bytes.NewBufferString(form.Encode()))
	require.NoError(t, err)
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp3, err := client.Do(req3)
	require.NoError(t, err)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	// 4. List completed — item appears
	resp4, err := http.Get(base + "/todo-completed")
	require.NoError(t, err)
	defer resp4.Body.Close()
	assert.Equal(t, http.StatusOK, resp4.StatusCode)
	var completed []todoResponse
	require.NoError(t, json.NewDecoder(resp4.Body).Decode(&completed))
	found = false
	for _, item := range completed {
		if item.Id == id {
			found = true
			assert.True(t, item.Completed)
			break
		}
	}
	assert.True(t, found, "updated item should appear in todo-completed")

	// 5. Delete
	req5, err := http.NewRequest(http.MethodDelete, base+"/todo/"+id, nil)
	require.NoError(t, err)
	resp5, err := client.Do(req5)
	require.NoError(t, err)
	defer resp5.Body.Close()
	assert.Equal(t, http.StatusOK, resp5.StatusCode)

	// 6. Get by id — 404 (delete persisted to DB)
	resp6, err := http.Get(base + "/todo/" + id)
	require.NoError(t, err)
	defer resp6.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp6.StatusCode, "deleted item must return 404")
}
