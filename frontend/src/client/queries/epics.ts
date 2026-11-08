import {queryOptions} from '@tanstack/vue-query'
import {projectTasksList} from '@/client/generated'
import {fetchAllPages} from './fetchAllPages'
import {queryClient} from '@/client/queryClient'
import {normalizeTask, type TaskResponse} from './tasks'

export const epicKeys = {
	all: ['epics'] as const,
	project: (projectId: number) => [...epicKeys.all, projectId] as const,
}

// Epics change with their tasks, which invalidate this list; the stale time only spares remounts.
const EPICS_STALE_TIME = 60 * 1000

export type EpicResponse = TaskResponse & {
	epic_progress: {total: number, done: number}
}

// Open epics first, then in creation order, the way they are listed in the epics panel.
function sortEpics(epics: EpicResponse[]): EpicResponse[] {
	return [...epics].sort((a, b) => Number(a.done) - Number(b.done) || a.id - b.id)
}

export function projectEpicsQuery(projectId: number) {
	return queryOptions({
		queryKey: epicKeys.project(projectId),
		queryFn: async ({signal}) => {
			const epics = await fetchAllPages(async page => (await projectTasksList({
				path: {project: projectId},
				query: {page, epics: 'only', expand: ['epic_progress']},
				signal,
			})).data)
			return sortEpics(epics.map(epic => {
				const task = normalizeTask(epic)
				return {
					...task,
					epic_progress: {
						total: epic.epic_progress?.total ?? 0,
						done: epic.epic_progress?.done ?? 0,
					},
				}
			}))
		},
		enabled: projectId > 0,
		staleTime: EPICS_STALE_TIME,
	})
}

export type EpicProgress = EpicResponse['epic_progress']

export function epicProgressPercent({total, done}: EpicProgress): number {
	return total === 0 ? 0 : Math.round(done / total * 100)
}

export function ensureProjectEpics(projectId: number): Promise<EpicResponse[]> {
	return queryClient.ensureQueryData(projectEpicsQuery(projectId))
}

export function findEpicByTitle(epics: EpicResponse[], title: string): EpicResponse | undefined {
	const wanted = title.toLowerCase()
	return epics.find(epic => epic.title.toLowerCase() === wanted)
}
