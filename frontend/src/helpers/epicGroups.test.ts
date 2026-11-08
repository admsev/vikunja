import {describe, expect, it} from 'vitest'

import {NO_EPIC} from '@/constants/epics'
import type {EpicResponse} from '@/client/queries/epics'
import {normalizeTask} from '@/client/queries/tasks'
import {epicGroupKey, groupTasksByEpic} from './epicGroups'

function task(id: number, epicId = 0) {
	return normalizeTask({id, title: `Task ${id}`, epic_id: epicId})
}

function epic(id: number): EpicResponse {
	return {...normalizeTask({id, title: `Epic ${id}`, is_epic: true}), epic_progress: {total: 0, done: 0}}
}

describe('groupTasksByEpic', () => {
	it('groups tasks in the order of the epics and puts tasks without an epic last', () => {
		const groups = groupTasksByEpic([task(1, 20), task(2), task(3, 10), task(4, 20)], [epic(10), epic(20)])

		expect(groups.map(group => [group.key, group.tasks.map(t => t.id)])).toEqual([
			['10', [3]],
			['20', [1, 4]],
			[NO_EPIC, [2]],
		])
	})

	it('treats tasks of unknown epics as without an epic and leaves out empty groups', () => {
		const groups = groupTasksByEpic([task(1, 99)], [epic(10)])

		expect(groups).toEqual([{key: NO_EPIC, epic: null, tasks: [task(1, 99)]}])
	})
})

describe('epicGroupKey', () => {
	it('keys an epic by its id and uses the no-epic key for 0', () => {
		expect(epicGroupKey(12)).toBe('12')
		expect(epicGroupKey(0)).toBe(NO_EPIC)
	})
})
