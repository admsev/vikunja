import {computed, type Ref} from 'vue'
import type {Task} from '@/client/generated'
import type {Filters} from '@/composables/useRouteFilters'
import {useQuery} from '@tanstack/vue-query'
import {allTasksQuery} from '@/client/queries/tasks'
import type {TaskFilterParams, TaskResponse} from '@/client/queries/tasks'
import {useCreateTaskMutation, useUpdateTaskMutation} from '@/client/queries/taskMutations'
import {useAuthStore} from '@/stores/auth'
import {useEpicFilter, withEpicFilter} from '@/composables/useEpicFilter'
import {useProjectEpics} from '@/composables/useProjectEpics'

export function useGanttTaskList<F extends Filters>(
	filters: Ref<F>,
	filterToApiParams: (filters: F) => TaskFilterParams,
	viewId: Ref<number>,
) {
	const auth = useAuthStore()
	const {epicFilter} = useEpicFilter()
	const query = useQuery(computed(() => {
		const params = filterToApiParams(filters.value)
		return allTasksQuery({
			project: filters.value.projectId,
			view: viewId.value,
			params: {
				...params,
				filter: withEpicFilter(params.filter ?? '', epicFilter.value, true),
				filter_timezone: auth.settings.timezone,
			},
		})
	}))
	// Epics often have no dates of their own; pull in the ones the shown tasks belong to so they
	// become summary rows spanning their tasks.
	const {epicsById} = useProjectEpics(() => filters.value.projectId)
	const tasks = computed(() => {
		const map = new Map<number, TaskResponse>((query.data.value ?? []).map(task => [task.id, task]))
		for (const task of [...map.values()]) {
			const epic = task.epic_id ? epicsById.value.get(task.epic_id) : undefined
			if (epic && !map.has(epic.id)) map.set(epic.id, epic)
		}
		return map
	})
	const create = useCreateTaskMutation()
	const update = useUpdateTaskMutation(true)
	return {
		tasks, isLoading: query.isFetching,
		loadTasks: async () => { await query.refetch() },
		addTask: (task: Task) => create.mutateAsync({...task, project_id: filters.value.projectId}),
		updateTask: async (task: Task) => {
			const current = tasks.value.get(task.id!)
			if (current) await update.mutateAsync({...current, ...task, id: task.id!})
		},
	}
}

export type UseGanttTaskListReturn = ReturnType<typeof useGanttTaskList>
