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

package migration

import (
	"slices"
	"testing"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/require"
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// The parts of tasks the migration must leave alone: a unique and a composite index next to plain ones.
type tasksBefore20261107121632 struct {
	ID        int64  `xorm:"bigint autoincr not null unique pk"`
	Title     string `xorm:"TEXT not null"`
	ProjectID int64  `xorm:"bigint INDEX not null unique(tasks_project_index) index(project_done_due_date)"`
	Done      bool   `xorm:"INDEX null index(project_done_due_date)"`
	Index     int64  `xorm:"bigint not null default 0 unique(tasks_project_index)"`
}

func (tasksBefore20261107121632) TableName() string { return "tasks" }

type tasksAfter20261107121632 struct {
	ID     int64 `xorm:"bigint autoincr not null unique pk"`
	IsEpic bool
	EpicID int64
}

func (tasksAfter20261107121632) TableName() string { return "tasks" }

func TestAddTaskEpics20261107121632(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	table := tasksBefore20261107121632{}
	t.Cleanup(func() {
		require.NoError(t, x.DropTables(table))
	})
	require.NoError(t, x.DropTables(table))
	require.NoError(t, x.Sync2(table))
	_, err = x.Insert(&tasksBefore20261107121632{ID: 1, Title: "existing", ProjectID: 1, Index: 1})
	require.NoError(t, err)

	before := tasksTable20261107121632(t, x)
	require.NotEmpty(t, before.Indexes)

	idx := slices.IndexFunc(migrations, func(m *xormigrate.Migration) bool { return m.ID == "20261107121632" })
	require.NotEqual(t, -1, idx)
	require.NoError(t, migrations[idx].Migrate(x))

	after := tasksTable20261107121632(t, x)
	require.NotNil(t, after.GetColumn("is_epic"))
	require.NotNil(t, after.GetColumn("epic_id"))
	for _, column := range before.ColumnsSeq() {
		require.NotNilf(t, after.GetColumn(column), "migration dropped column %s", column)
	}
	for name, index := range before.Indexes {
		preserved, found := after.Indexes[name]
		require.Truef(t, found, "migration dropped index %s", name)
		require.Equal(t, index.Type, preserved.Type)
		require.Equal(t, index.Cols, preserved.Cols)
	}

	// Existing tasks become regular tasks without an epic.
	got := &tasksAfter20261107121632{}
	found, err := x.ID(1).Get(got)
	require.NoError(t, err)
	require.True(t, found)
	require.False(t, got.IsEpic)
	require.Zero(t, got.EpicID)
}

func tasksTable20261107121632(t *testing.T, x *xorm.Engine) *schemas.Table {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	for _, table := range tables {
		if table.Name == "tasks" {
			return table
		}
	}
	t.Fatal("tasks table not found")
	return nil
}
