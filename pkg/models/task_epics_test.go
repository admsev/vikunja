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

package models

import (
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// makeEpic turns fixture task 1 (project 1) into an epic and puts tasks 2 and 3 into it.
func makeEpic(t *testing.T, s *xorm.Session, u *user.User) {
	t.Helper()
	epic := &Task{ID: 1, IsEpic: true}
	require.NoError(t, epic.updateSingleTask(s, u, []string{"is_epic"}))

	for _, id := range []int64{2, 3} {
		task := &Task{ID: id, EpicID: 1}
		require.NoError(t, task.updateSingleTask(s, u, []string{"epic_id"}))
	}
}

func TestTaskEpic_Assign(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("task joins an epic of its project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 1, "is_epic": true, "epic_id": 0}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 1}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 3, "epic_id": 1}, false)
	})
	t.Run("full update keeps unrelated fields", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		task, err := GetTaskByIDSimple(s, 2)
		require.NoError(t, err)
		task.Title = "renamed"
		require.NoError(t, task.Update(s, u))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "title": "renamed", "epic_id": 1}, false)
	})
	t.Run("rejects a task that is not an epic", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		err := (&Task{ID: 2, EpicID: 3}).updateSingleTask(s, u, []string{"epic_id"})
		require.Error(t, err)
		assert.True(t, IsErrInvalidTaskEpic(err))
	})
	t.Run("rejects an epic from another project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// Task 13 is in project 2, which user 1 can't write; make it an epic directly.
		setTaskColumnsForTest(t, 13, map[string]any{"is_epic": true})
		err := (&Task{ID: 2, EpicID: 13}).updateSingleTask(s, u, []string{"epic_id"})
		require.Error(t, err)
		assert.True(t, IsErrInvalidTaskEpic(err))
	})
	t.Run("rejects a nonexistent epic", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		err := (&Task{ID: 2, EpicID: 99999}).updateSingleTask(s, u, []string{"epic_id"})
		require.Error(t, err)
		assert.True(t, IsErrInvalidTaskEpic(err))
	})
	t.Run("rejects nesting epics", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		require.NoError(t, (&Task{ID: 4, IsEpic: true}).updateSingleTask(s, u, []string{"is_epic"}))
		err := (&Task{ID: 4, EpicID: 1}).updateSingleTask(s, u, []string{"epic_id"})
		require.Error(t, err)
		assert.True(t, IsErrInvalidTaskEpic(err))
	})
	t.Run("create into an epic", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		task := &Task{Title: "new", ProjectID: 1, EpicID: 1}
		require.NoError(t, task.Create(s, u))
		require.NoError(t, s.Commit())
		db.AssertExists(t, "tasks", map[string]interface{}{"id": task.ID, "epic_id": 1}, false)
	})
	t.Run("full update from a client without epic fields keeps them", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		require.NoError(t, (&Task{ID: 1, Title: "renamed epic", ProjectID: 1}).Update(s, u))
		require.NoError(t, (&Task{ID: 2, Title: "renamed task", ProjectID: 1}).Update(s, u))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 1, "is_epic": true}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 1}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 3, "epic_id": 1}, false)
	})
	t.Run("full update with provided epic fields applies them", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		task := &Task{ID: 2, Title: "task", ProjectID: 1}
		task.MarkEpicFieldsProvided()
		require.NoError(t, task.Update(s, u))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 0}, false)
	})
	t.Run("create rejects an invalid epic", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		err := (&Task{Title: "new", ProjectID: 1, EpicID: 2}).Create(s, u)
		require.Error(t, err)
		assert.True(t, IsErrInvalidTaskEpic(err))
	})
}

func TestTaskEpic_Lifecycle(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("converting an epic back detaches its tasks", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		require.NoError(t, (&Task{ID: 1, IsEpic: false}).updateSingleTask(s, u, []string{"is_epic"}))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 1, "is_epic": false}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 0}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 3, "epic_id": 0}, false)
	})
	t.Run("trashing an epic keeps its tasks attached", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		makeEpic(t, s, u)
		require.NoError(t, s.Commit())
		s.Close()

		trashTaskForTest(t, 1, u)

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 1}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 3, "epic_id": 1}, false)
		s = db.NewSession()
		defer s.Close()
		err := (&Task{ID: 4, EpicID: 1}).updateSingleTask(s, u, []string{"epic_id"})
		assert.True(t, IsErrInvalidTaskEpic(err), "a trashed epic takes no new tasks")
	})
	t.Run("restoring an epic brings its tasks back and keeps it off the boards", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		makeEpic(t, s, u)
		require.NoError(t, s.Commit())
		s.Close()

		trashTaskForTest(t, 1, u)
		_, err := restoreTaskForTest(t, 1, u)
		require.NoError(t, err)

		s = db.NewSession()
		defer s.Close()
		epic := &Task{ID: 1, Expand: []TaskCollectionExpandable{TaskCollectionExpandEpicProgress}}
		require.NoError(t, epic.ReadOne(s, u))
		require.NotNil(t, epic.EpicProgress)
		assert.Equal(t, int64(2), epic.EpicProgress.Total)
		db.AssertMissing(t, "task_buckets", map[string]interface{}{"task_id": 1})
	})
	t.Run("deleting an epic permanently releases its tasks", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		makeEpic(t, s, u)
		require.NoError(t, s.Commit())
		s.Close()

		trashTaskForTest(t, 1, u)
		s = db.NewSession()
		defer s.Close()
		epic := &Task{ID: 1}
		can, err := epic.CanDeletePermanently(s, u)
		require.NoError(t, err)
		require.True(t, can)
		require.NoError(t, epic.DeletePermanently(s, u))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 0}, false)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 3, "epic_id": 0}, false)
	})
	t.Run("moving a task to another project drops its epic", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		require.NoError(t, (&Task{ID: 2, ProjectID: 2}).updateSingleTask(s, u, []string{"project_id"}))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "project_id": 2, "epic_id": 0}, false)
	})
	t.Run("turning a task into an epic drops its own epic", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		require.NoError(t, (&Task{ID: 2, IsEpic: true}).updateSingleTask(s, u, []string{"is_epic"}))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "is_epic": true, "epic_id": 0}, false)
	})
}

func TestTaskEpic_ReadAndFilter(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("expand epic and epic_progress", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)

		child := &Task{ID: 2, Expand: []TaskCollectionExpandable{TaskCollectionExpandEpic}}
		require.NoError(t, child.ReadOne(s, u))
		require.NotNil(t, child.Epic)
		assert.Equal(t, int64(1), child.Epic.ID)
		assert.Equal(t, "task #1", child.Epic.Title)
		assert.NotEmpty(t, child.Epic.Identifier)

		epic := &Task{ID: 1, Expand: []TaskCollectionExpandable{TaskCollectionExpandEpicProgress}}
		require.NoError(t, epic.ReadOne(s, u))
		require.NotNil(t, epic.EpicProgress)
		// Fixture task 2 is done, task 3 is not.
		assert.Equal(t, int64(2), epic.EpicProgress.Total)
		assert.Equal(t, int64(1), epic.EpicProgress.Done)
	})
	t.Run("filter by epic and epic scope", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)

		ids := func(filter string, scope EpicScope) []int64 {
			tc := &TaskCollection{ProjectID: 1, Filter: filter, EpicScope: scope}
			res, _, _, err := tc.ReadAll(s, u, "", 1, 50)
			require.NoError(t, err)
			tasks, ok := res.([]*Task)
			require.True(t, ok)
			out := []int64{}
			for _, task := range tasks {
				out = append(out, task.ID)
			}
			return out
		}

		assert.ElementsMatch(t, []int64{2, 3}, ids("epic = 1", EpicScopeInclude))
		assert.ElementsMatch(t, []int64{2, 3}, ids("epic_id in 1", EpicScopeInclude))
		assert.Equal(t, []int64{1}, ids("", EpicScopeOnly))
		assert.NotContains(t, ids("", EpicScopeExclude), int64(1))
		assert.NotContains(t, ids("epic = 0", EpicScopeInclude), int64(2))
		// The scope is its own condition, not text around the filter, so a filter that closes
		// its parentheses is rejected instead of widening the scope.
		_, _, _, err := (&TaskCollection{ProjectID: 1, Filter: "done = true) || (id > 0", EpicScope: EpicScopeExclude}).ReadAll(s, u, "", 1, 50)
		require.Error(t, err)
	})
}

func TestTaskEpic_Buckets(t *testing.T) {
	u := &user.User{ID: 1}
	const kanbanView = 4 // the manual kanban view of fixture project 1

	t.Run("a new epic is not put into a bucket", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		epic := &Task{Title: "epic", ProjectID: 1, IsEpic: true}
		require.NoError(t, epic.Create(s, u))
		require.NoError(t, s.Commit())

		db.AssertMissing(t, "task_buckets", map[string]interface{}{"task_id": epic.ID})
	})
	t.Run("converting moves a task out of and back into the buckets", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		require.NoError(t, (&Task{ID: 1, IsEpic: true}).updateSingleTask(s, u, []string{"is_epic"}))
		has, err := s.Exist(&TaskBucket{TaskID: 1, ProjectViewID: kanbanView})
		require.NoError(t, err)
		assert.False(t, has)

		require.NoError(t, (&Task{ID: 1, IsEpic: false}).updateSingleTask(s, u, []string{"is_epic"}))
		has, err = s.Exist(&TaskBucket{TaskID: 1, ProjectViewID: kanbanView})
		require.NoError(t, err)
		assert.True(t, has)
	})
}

func TestTaskEpic_ProjectDuplicate(t *testing.T) {
	files.InitTestFileFixtures(t)
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	makeEpic(t, s, u)
	pd := &ProjectDuplicate{ProjectID: 1}
	can, err := pd.CanCreate(s, u)
	require.NoError(t, err)
	require.True(t, can)
	require.NoError(t, pd.Create(s, u))

	epic := &Task{}
	has, err := s.Where("project_id = ? AND is_epic = ?", pd.Project.ID, true).Get(epic)
	require.NoError(t, err)
	require.True(t, has)
	count, err := s.Where("project_id = ? AND epic_id = ?", pd.Project.ID, epic.ID).Count(&Task{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestTaskEpic_Duplicate(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("a duplicated task stays in its epic", func(t *testing.T) {
		files.InitTestFileFixtures(t)
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		td := &TaskDuplicate{TaskID: 2}
		require.NoError(t, td.Create(s, u))
		require.NoError(t, s.Commit())

		assert.Equal(t, int64(1), td.Task.EpicID)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": td.Task.ID, "epic_id": 1}, false)
	})
	t.Run("a duplicated epic is an epic", func(t *testing.T) {
		files.InitTestFileFixtures(t)
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		makeEpic(t, s, u)
		td := &TaskDuplicate{TaskID: 1}
		require.NoError(t, td.Create(s, u))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": td.Task.ID, "is_epic": true, "epic_id": 0}, false)
		// The original keeps its tasks; the copy starts empty.
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 1}, false)
	})
}

func TestTaskEpic_Regressions(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("converting an epic back also releases its trashed tasks", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		makeEpic(t, s, u)
		require.NoError(t, s.Commit())
		s.Close()

		trashTaskForTest(t, 2, u)
		trashTaskForTest(t, 3, u)
		s = db.NewSession()
		defer s.Close()
		require.NoError(t, (&Task{ID: 1, IsEpic: false}).updateSingleTask(s, u, []string{"is_epic"}))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 0}, true)
		db.AssertExists(t, "tasks", map[string]interface{}{"id": 3, "epic_id": 0}, true)
	})
	t.Run("restoring a task drops an epic that is no longer one", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		trashTaskForTest(t, 2, u)
		// Task 4 is a regular task: the stored link went stale while task 2 was in the trash.
		setTaskColumnsForTest(t, 2, map[string]any{"epic_id": 4})

		_, err := restoreTaskForTest(t, 2, u)
		require.NoError(t, err)

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 0}, false)
	})
	t.Run("restoring a task keeps an epic that is only in the trash", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		makeEpic(t, s, u)
		require.NoError(t, s.Commit())
		s.Close()

		trashTaskForTest(t, 2, u)
		trashTaskForTest(t, 1, u)
		_, err := restoreTaskForTest(t, 2, u)
		require.NoError(t, err)

		db.AssertExists(t, "tasks", map[string]interface{}{"id": 2, "epic_id": 1}, false)
	})
	t.Run("an epic in another project reads like a missing one", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		setTaskColumnsForTest(t, 13, map[string]any{"is_epic": true})
		foreign := (&Task{ID: 2, EpicID: 13}).updateSingleTask(s, u, []string{"epic_id"})
		missing := (&Task{ID: 2, EpicID: 99999}).updateSingleTask(s, u, []string{"epic_id"})

		var foreignErr, missingErr ErrInvalidTaskEpic
		require.ErrorAs(t, foreign, &foreignErr)
		require.ErrorAs(t, missing, &missingErr)
		assert.Equal(t, missingErr.HTTPError().Message, foreignErr.HTTPError().Message)
	})
	t.Run("the epic expand ignores a link to another project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		setTaskColumnsForTest(t, 13, map[string]any{"is_epic": true})
		setTaskColumnsForTest(t, 2, map[string]any{"epic_id": 13})
		task := &Task{ID: 2, Expand: []TaskCollectionExpandable{TaskCollectionExpandEpic}}
		require.NoError(t, task.ReadOne(s, u))
		assert.Nil(t, task.Epic)
	})
	t.Run("an epic cannot be put into a bucket", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		makeEpic(t, s, u)
		require.NoError(t, s.Commit())
		s.Close()

		s = db.NewSession()
		defer s.Close()
		err := (&TaskBucket{TaskID: 1, BucketID: 1, ProjectViewID: 4, ProjectID: 1}).Update(s, u)
		assert.True(t, IsErrEpicCannotBeInBucket(err))
	})
}

func TestTaskEpic_Permissions(t *testing.T) {
	t.Run("a user without access to the project cannot assign its epics", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		makeEpic(t, s, &user.User{ID: 1})
		require.NoError(t, s.Commit())
		s.Close()

		s = db.NewSession()
		defer s.Close()
		can, err := (&Task{ID: 4, EpicID: 1}).CanUpdate(s, &user.User{ID: 13})
		require.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("a read-only link share cannot change a task's epic", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		share := &LinkSharing{ID: 1, Hash: "test", ProjectID: 1, Permission: PermissionRead, SharedByID: 1}
		can, err := (&Task{ID: 2, EpicID: 1}).CanUpdate(s, share)
		require.NoError(t, err)
		assert.False(t, can)
	})
	t.Run("a writer of the project can assign its epics", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		can, err := (&Task{ID: 2, EpicID: 1}).CanUpdate(s, &user.User{ID: 1})
		require.NoError(t, err)
		assert.True(t, can)
	})
}
