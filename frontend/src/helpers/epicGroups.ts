import {NO_EPIC} from '@/constants/epics'
import type {EpicResponse} from '@/client/queries/epics'
import type {TaskResponse} from '@/client/queries/tasks'

export type EpicGroup = {
	key: string,
	epic: EpicResponse | null,
	tasks: TaskResponse[],
}

export function epicGroupKey(epicId: number): string {
	return epicId > 0 ? String(epicId) : NO_EPIC
}

/**
 * Splits tasks into one group per epic, in the order of `epics`, followed by the tasks without an epic.
 * Tasks whose epic is not in `epics` count as without an epic. Empty groups are left out.
 */
export function groupTasksByEpic(tasks: TaskResponse[], epics: EpicResponse[]): EpicGroup[] {
	const known = new Map(epics.map(epic => [epic.id, epic]))
	const byKey = new Map<string, TaskResponse[]>()
	for (const task of tasks) {
		const key = epicGroupKey(known.has(task.epic_id ?? 0) ? task.epic_id ?? 0 : 0)
		byKey.set(key, [...(byKey.get(key) ?? []), task])
	}

	const groups: EpicGroup[] = epics
		.filter(epic => byKey.has(epicGroupKey(epic.id)))
		.map(epic => ({key: epicGroupKey(epic.id), epic, tasks: byKey.get(epicGroupKey(epic.id))!}))
	const withoutEpic = byKey.get(NO_EPIC)
	if (withoutEpic) groups.push({key: NO_EPIC, epic: null, tasks: withoutEpic})
	return groups
}
