import {computed} from 'vue'
import {useRouteQuery} from '@vueuse/router'

import {NO_EPIC} from '@/constants/epics'
import type {TaskFilterParams} from '@/client/queries/tasks'

// The epic a project view is narrowed to, kept in the `epic` route query so it survives
// reloads and switching between views: an epic id, or NO_EPIC for tasks without an epic.
export type EpicFilter = number | typeof NO_EPIC | null

export function parseEpicFilter(raw: unknown): EpicFilter {
	if (raw === NO_EPIC) return NO_EPIC
	const id = Number(raw)
	return Number.isInteger(id) && id > 0 ? id : null
}

/**
 * Combines a view's filter with the epic filter in the backend filter syntax.
 * includeEpic keeps the epic itself in the result, for views that show epics next to their tasks.
 */
export function withEpicFilter(filter: string, epic: EpicFilter, includeEpic = false): string {
	if (epic === null) return filter
	let cond = `epic = ${epic === NO_EPIC ? 0 : epic}`
	if (includeEpic && epic !== NO_EPIC) cond = `(${cond} || id = ${epic})`
	return filter ? `(${filter}) && ${cond}` : cond
}

/**
 * Filter params for views that list regular tasks and manage epics in the epics panel.
 * Pseudo projects (favorites, saved filters) have no panel and keep their epics.
 */
export function regularTaskParams(projectId: number, filter: string, epic: EpicFilter): Pick<TaskFilterParams, 'filter' | 'epics'> {
	return projectId > 0
		? {filter: withEpicFilter(filter, epic), epics: 'exclude'}
		: {filter}
}

export function useEpicFilter() {
	const raw = useRouteQuery<string | undefined>('epic')
	const epicFilter = computed<EpicFilter>({
		get: () => parseEpicFilter(raw.value),
		set: value => { raw.value = value === null ? undefined : String(value) },
	})

	function toggleEpicFilter(value: Exclude<EpicFilter, null>) {
		epicFilter.value = epicFilter.value === value ? null : value
	}

	return {epicFilter, toggleEpicFilter}
}
