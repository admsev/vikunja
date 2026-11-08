// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package webtests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/models"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// epicTask is the part of a task response these tests read.
type epicTask struct {
	ID           int64                `json:"id"`
	IsEpic       bool                 `json:"is_epic"`
	EpicID       int64                `json:"epic_id"`
	Epic         *models.EpicSummary  `json:"epic"`
	EpicProgress *models.EpicProgress `json:"epic_progress"`
}

type epicTaskList struct {
	Items []epicTask `json:"items"`
}

func epicRequest(t *testing.T, e *echo.Echo, token, method, path, body string) (int, []byte) {
	t.Helper()
	contentType := ""
	if method == http.MethodPatch {
		contentType = "application/merge-patch+json"
	}
	rec := humaRequest(t, e, method, path, body, token, contentType)
	return rec.Code, rec.Body.Bytes()
}

func decodeEpicJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(body, &v), "body: %s", body)
	return v
}

func listedIDs(list epicTaskList) []int64 {
	ids := make([]int64, 0, len(list.Items))
	for _, task := range list.Items {
		ids = append(ids, task.ID)
	}
	return ids
}

// TestHumaTaskEpics covers epics end to end on project 1, owned by user1.
func TestHumaTaskEpics(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	token := humaTokenFor(t, &testuser1)

	code, body := epicRequest(t, e, token, http.MethodPost, "/api/v2/projects/1/tasks", `{"title":"Launch plan","is_epic":true}`)
	require.Equal(t, http.StatusCreated, code, "body: %s", body)
	epic := decodeEpicJSON[epicTask](t, body)
	require.True(t, epic.IsEpic)

	code, body = epicRequest(t, e, token, http.MethodPost, "/api/v2/projects/1/tasks", fmt.Sprintf(`{"title":"Press release","epic_id":%d}`, epic.ID))
	require.Equal(t, http.StatusCreated, code, "body: %s", body)
	child := decodeEpicJSON[epicTask](t, body)
	require.Equal(t, epic.ID, child.EpicID)

	t.Run("rejects invalid epics with 4043", func(t *testing.T) {
		for name, payload := range map[string]string{
			"missing epic":     `{"title":"x","epic_id":999999}`,
			"not an epic":      fmt.Sprintf(`{"title":"x","epic_id":%d}`, child.ID),
			"epic inside epic": fmt.Sprintf(`{"title":"x","is_epic":true,"epic_id":%d}`, epic.ID),
		} {
			t.Run(name, func(t *testing.T) {
				code, body := epicRequest(t, e, token, http.MethodPost, "/api/v2/projects/1/tasks", payload)
				assert.Equal(t, http.StatusBadRequest, code)
				assert.Contains(t, string(body), fmt.Sprintf(`"code":%d`, models.ErrCodeInvalidTaskEpic))
			})
		}
	})
	t.Run("lists epics only, or excludes them", func(t *testing.T) {
		_, body := epicRequest(t, e, token, http.MethodGet, "/api/v2/projects/1/tasks?epics=only&expand=epic_progress", "")
		epics := decodeEpicJSON[epicTaskList](t, body)
		require.Len(t, epics.Items, 1)
		assert.Equal(t, epic.ID, epics.Items[0].ID)
		require.NotNil(t, epics.Items[0].EpicProgress)
		assert.Equal(t, int64(1), epics.Items[0].EpicProgress.Total)

		_, body = epicRequest(t, e, token, http.MethodGet, "/api/v2/projects/1/tasks?epics=exclude&per_page=100", "")
		assert.NotContains(t, listedIDs(decodeEpicJSON[epicTaskList](t, body)), epic.ID)
	})
	t.Run("filters by epic and expands the epic summary", func(t *testing.T) {
		_, body := epicRequest(t, e, token, http.MethodGet, fmt.Sprintf("/api/v2/projects/1/tasks?filter=epic+%%3D+%d&expand=epic", epic.ID), "")
		list := decodeEpicJSON[epicTaskList](t, body)
		require.Equal(t, []int64{child.ID}, listedIDs(list))
		require.NotNil(t, list.Items[0].Epic)
		assert.Equal(t, "Launch plan", list.Items[0].Epic.Title)
	})
	t.Run("bulk assigns tasks to the epic", func(t *testing.T) {
		code, body := epicRequest(t, e, token, http.MethodPut, "/api/v2/tasks/bulk", fmt.Sprintf(`{"task_ids":[1,2],"fields":["epic_id"],"values":{"epic_id":%d}}`, epic.ID))
		require.Equal(t, http.StatusOK, code, "body: %s", body)

		_, body = epicRequest(t, e, token, http.MethodGet, fmt.Sprintf("/api/v2/projects/1/tasks?filter=epic+%%3D+%d", epic.ID), "")
		assert.ElementsMatch(t, []int64{1, 2, child.ID}, listedIDs(decodeEpicJSON[epicTaskList](t, body)))
	})
	t.Run("PATCH takes a task out of its epic", func(t *testing.T) {
		code, body := epicRequest(t, e, token, http.MethodPatch, fmt.Sprintf("/api/v2/tasks/%d", child.ID), `{"epic_id":0}`)
		require.Equal(t, http.StatusOK, code, "body: %s", body)
		assert.Zero(t, decodeEpicJSON[epicTask](t, body).EpicID)
	})
	t.Run("a user without access cannot change a task's epic", func(t *testing.T) {
		code, _ := epicRequest(t, e, humaTokenFor(t, &testuser2), http.MethodPatch, "/api/v2/tasks/3", fmt.Sprintf(`{"epic_id":%d}`, epic.ID))
		assert.Equal(t, http.StatusForbidden, code)
	})
	t.Run("an epic cannot be put into a bucket", func(t *testing.T) {
		code, body := epicRequest(t, e, token, http.MethodPut, "/api/v2/projects/1/views/4/buckets/1/tasks", fmt.Sprintf(`{"task_id":%d}`, epic.ID))
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, string(body), fmt.Sprintf(`"code":%d`, models.ErrCodeEpicCannotBeInBucket))
	})
}
