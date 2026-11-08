import {computed, toValue, type MaybeRefOrGetter} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import {allTasksQuery} from '@/client/queries/tasks'
import {withEpicFilter} from '@/composables/useEpicFilter'

type EpicRef = {id: number, project_id: number}

// The tasks of an epic, open ones first. A null epic disables the query.
export function useEpicTasks(epic: MaybeRefOrGetter<EpicRef | null>) {
	const query = useQuery(computed(() => {
		const current = toValue(epic)
		return {
			...allTasksQuery({
				project: current?.project_id ?? 0,
				params: {
					filter: withEpicFilter('', current?.id ?? null),
					epics: 'exclude',
					sort_by: ['done', 'id'],
					order_by: ['asc', 'asc'],
				},
			}),
			enabled: current !== null,
		}
	}))
	const tasks = computed(() => query.data.value ?? [])
	return {
		tasks,
		isPending: query.isPending,
		progress: computed(() => ({
			done: tasks.value.filter(task => task.done).length,
			total: tasks.value.length,
		})),
	}
}
