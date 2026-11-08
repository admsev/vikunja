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
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// EpicScope selects whether a task list contains regular tasks, epics or both.
type EpicScope string

const (
	EpicScopeInclude EpicScope = "include"
	EpicScopeExclude EpicScope = "exclude"
	EpicScopeOnly    EpicScope = "only"
)

// EpicSummary is the part of an epic shown on the tasks belonging to it.
type EpicSummary struct {
	ID         int64  `json:"id" doc:"The id of the epic."`
	Title      string `json:"title" doc:"The title of the epic."`
	HexColor   string `json:"hex_color" doc:"The epic color as a hex string without the leading '#'."`
	Identifier string `json:"identifier" doc:"The textual identifier of the epic, e.g. \"PROJ-12\"."`
	Done       bool   `json:"done" doc:"Whether the epic is done."`
}

// EpicProgress counts the tasks of an epic.
type EpicProgress struct {
	Total int64 `json:"total" doc:"The number of tasks in this epic."`
	Done  int64 `json:"done" doc:"The number of done tasks in this epic."`
}

// MarkEpicFieldsProvided tells Update that is_epic and epic_id came from the client.
// Full updates from clients that predate epics (v1, CalDAV) leave them unset and keep the stored values.
func (t *Task) MarkEpicFieldsProvided() {
	t.epicFieldsProvided = true
}

func invalidEpic(t *Task, reason string) error {
	return ErrInvalidTaskEpic{TaskID: t.ID, EpicID: t.EpicID, Reason: reason}
}

// errEpicNotFound covers missing epics and tasks in other projects alike, so the answer
// reveals nothing about projects the user may not be able to read.
func errEpicNotFound(t *Task) error {
	return invalidEpic(t, "the epic does not exist in this project")
}

// checkEpicFor checks that epic, nil when it does not exist, may hold task t.
func checkEpicFor(t *Task, epic *Task) error {
	switch {
	case epic == nil || epic.ProjectID != t.ProjectID:
		return errEpicNotFound(t)
	case !epic.IsEpic:
		return invalidEpic(t, "the referenced task is not an epic")
	}
	return nil
}

// validateEpicForTask checks that t.EpicID refers to an epic the task may belong to.
func validateEpicForTask(s *xorm.Session, t *Task) error {
	switch {
	case t.EpicID == 0:
		return nil
	case t.IsEpic:
		return invalidEpic(t, "an epic cannot belong to another epic")
	case t.EpicID == t.ID:
		return invalidEpic(t, "a task cannot be its own epic")
	}

	// Locked so a concurrent conversion of the epic back to a task can't strand this one.
	epic, err := GetTaskSimple(lockingSession(s), &Task{ID: t.EpicID})
	if IsErrTaskDoesNotExist(err) {
		return checkEpicFor(t, nil)
	}
	if err != nil {
		return err
	}
	return checkEpicFor(t, &epic)
}

// validateEpicsForTasks validates the epics of a batch of new tasks, loading each epic once.
func validateEpicsForTasks(s *xorm.Session, tasks []*Task) error {
	epicIDs := []int64{}
	for _, t := range tasks {
		if t.EpicID != 0 && !t.IsEpic {
			epicIDs = append(epicIDs, t.EpicID)
		}
	}
	epics := map[int64]*Task{}
	if len(epicIDs) > 0 {
		if err := lockingSession(s).In("id", epicIDs).Find(&epics); err != nil {
			return err
		}
	}

	for i, t := range tasks {
		var err error
		switch {
		case t.EpicID == 0:
		case t.IsEpic:
			err = invalidEpic(t, "an epic cannot belong to another epic")
		default:
			err = checkEpicFor(t, epics[t.EpicID])
		}
		if err != nil {
			return ErrInvalidTaskInBulkCreation{Index: i, Err: err}
		}
	}
	return nil
}

// applyEpicChanges reconciles is_epic and epic_id of an update before it is saved and reports
// whether epic_id changed, so the caller writes the column even when it was not asked for.
func applyEpicChanges(s *xorm.Session, a web.Auth, t *Task, ot *Task) (epicIDChanged bool, err error) {
	movedProject := t.ProjectID != ot.ProjectID
	becameEpic := t.IsEpic && !ot.IsEpic

	// A task keeps its epic only within the epic's project, and an epic has no epic of its own.
	if t.EpicID != 0 && t.EpicID == ot.EpicID && (movedProject || becameEpic) {
		t.EpicID = 0
	}

	if t.EpicID != ot.EpicID || t.IsEpic != ot.IsEpic || movedProject {
		if err := validateEpicForTask(s, t); err != nil {
			return false, err
		}
	}

	// An epic that stops being one, or leaves its project, releases its tasks.
	if ot.IsEpic && (!t.IsEpic || movedProject) {
		if err := detachEpicTasks(s, a, ot.ID); err != nil {
			return false, err
		}
	}

	return t.EpicID != ot.EpicID, nil
}

// detachEpicTasks removes all tasks from an epic, trashed ones included. The tasks themselves are kept.
func detachEpicTasks(s *xorm.Session, a web.Auth, epicID int64) error {
	// Only the live tasks are announced; trashed ones are invisible until restored.
	tasks := []*Task{}
	if err := s.Where(builder.Eq{taskPropertyEpicID: epicID}).Find(&tasks); err != nil {
		return err
	}
	if err := clearEpicID(s, epicID); err != nil {
		return err
	}

	doer := doerFromAuth(s, a)
	for _, t := range tasks {
		t.EpicID = 0
		events.DispatchOnCommit(s, &TaskUpdatedEvent{Task: t, Doer: doer})
	}
	return nil
}

// releaseStaleEpic drops the epic of a restored task when that epic is gone for good, left the
// project or is no longer an epic. A trashed epic still counts: restoring it brings the task back.
func releaseStaleEpic(s *xorm.Session, t *Task) error {
	if t.EpicID == 0 {
		return nil
	}
	epic := &Task{}
	has, err := s.Unscoped().ID(t.EpicID).Get(epic)
	if err != nil || (has && epic.IsEpic && epic.ProjectID == t.ProjectID) {
		return err
	}
	t.EpicID = 0
	_, err = s.ID(t.ID).Cols(taskPropertyEpicID).NoAutoTime().Update(&Task{EpicID: 0})
	return err
}

// clearEpicID takes every task, trashed ones included, out of an epic.
func clearEpicID(s *xorm.Session, epicID int64) error {
	_, err := s.
		Unscoped().
		Where(builder.Eq{taskPropertyEpicID: epicID}).
		Cols(taskPropertyEpicID).
		Update(&Task{EpicID: 0})
	return err
}

// syncEpicBuckets keeps epics out of kanban buckets: boards only show regular tasks, and bucket
// limits and counts would otherwise include invisible epics.
// kanbanViews are the project's manual kanban views.
func syncEpicBuckets(s *xorm.Session, a web.Auth, t *Task, wasEpic bool, kanbanViews []*ProjectView) error {
	switch {
	case t.IsEpic == wasEpic:
		return nil
	case t.IsEpic:
		_, err := s.Where("task_id = ?", t.ID).Delete(&TaskBucket{})
		return err
	}
	return addTaskToDefaultBuckets(s, a, t, kanbanViews)
}

// addTaskToDefaultBuckets puts a task without buckets into the done bucket of each view if it is
// done, otherwise into the default one.
func addTaskToDefaultBuckets(s *xorm.Session, a web.Auth, t *Task, views []*ProjectView) error {
	for _, view := range views {
		bucketID := view.DoneBucketID
		if bucketID == 0 || !t.Done {
			var err error
			bucketID, err = getDefaultBucketID(s, view)
			if err != nil {
				return err
			}
		}

		err := updateTaskBucket(s, a, &TaskBucket{
			BucketID:      bucketID,
			TaskID:        t.ID,
			ProjectViewID: view.ID,
			ProjectID:     t.ProjectID,
		})
		if err != nil {
			return err
		}

		tp, err := calculateNewPositionForTask(s, a, t, view)
		if err != nil {
			return err
		}
		if err := updateTaskPosition(s, a, tp); err != nil {
			return err
		}
	}
	return nil
}

func addEpicsToTasks(s *xorm.Session, taskMap map[int64]*Task, projects map[int64]*Project) error {
	epicIDs := []int64{}
	for _, t := range taskMap {
		if t.EpicID != 0 {
			epicIDs = append(epicIDs, t.EpicID)
		}
	}
	if len(epicIDs) == 0 {
		return nil
	}

	epics := []*Task{}
	err := s.
		Where(builder.In("id", epicIDs)).
		Cols("id", "title", "hex_color", "done", "project_id", "index", taskPropertyIsEpic).
		Find(&epics)
	if err != nil {
		return err
	}

	summaries := make(map[int64]*EpicSummary, len(epics))
	epicProjects := make(map[int64]int64, len(epics))
	for _, e := range epics {
		if !e.IsEpic {
			continue
		}
		epicProjects[e.ID] = e.ProjectID
		e.setIdentifier(projects[e.ProjectID])
		summaries[e.ID] = &EpicSummary{
			ID:         e.ID,
			Title:      e.Title,
			HexColor:   e.HexColor,
			Identifier: e.Identifier,
			Done:       e.Done,
		}
	}

	for _, t := range taskMap {
		if summary, has := summaries[t.EpicID]; has && epicProjects[t.EpicID] == t.ProjectID {
			t.Epic = summary
		}
	}
	return nil
}

// countTasksPerEpic counts the non-deleted tasks of the given epics that match cond.
func countTasksPerEpic(s *xorm.Session, epicIDs []int64, cond builder.Cond) (map[int64]int64, error) {
	type epicCount struct {
		EpicID int64 `xorm:"epic_id"`
		Count  int64 `xorm:"count"`
	}
	counts := []epicCount{}
	err := s.
		Table("tasks").
		Select("epic_id, COUNT(*) AS count").
		Where(builder.And(builder.In(taskPropertyEpicID, epicIDs), taskNotDeletedCond("tasks"), cond)).
		GroupBy(taskPropertyEpicID).
		Find(&counts)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int64, len(counts))
	for _, c := range counts {
		result[c.EpicID] = c.Count
	}
	return result, nil
}

func addEpicProgressToTasks(s *xorm.Session, taskMap map[int64]*Task) error {
	epicIDs := []int64{}
	for _, t := range taskMap {
		if t.IsEpic {
			epicIDs = append(epicIDs, t.ID)
		}
	}
	if len(epicIDs) == 0 {
		return nil
	}

	totals, err := countTasksPerEpic(s, epicIDs, nil)
	if err != nil {
		return err
	}
	done, err := countTasksPerEpic(s, epicIDs, builder.Eq{"done": true})
	if err != nil {
		return err
	}

	for _, id := range epicIDs {
		taskMap[id].EpicProgress = &EpicProgress{Total: totals[id], Done: done[id]}
	}
	return nil
}

// cond restricts a task query on tableAlias to the scope. EpicScopeInclude, the default, adds nothing.
func (scope EpicScope) cond(tableAlias string) builder.Cond {
	switch scope {
	case EpicScopeExclude:
		return builder.Eq{tableAlias + "." + taskPropertyIsEpic: false}
	case EpicScopeOnly:
		return builder.Eq{tableAlias + "." + taskPropertyIsEpic: true}
	case EpicScopeInclude:
	}
	return nil
}
