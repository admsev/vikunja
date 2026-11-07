import {useAssignTaskToEpicMutation, type EpicRef} from '@/client/queries/taskMutations'
import {useTaskDragState} from '@/composables/useTaskDragState'

// The epics panel marks its drop targets with data-epic-id ("0" takes a task out of its epic).
function findEpicDropTarget(x: number, y: number): EpicRef | null {
	if (!Number.isFinite(x) || !Number.isFinite(y) || typeof document.elementsFromPoint !== 'function') return null

	for (const el of document.elementsFromPoint(x, y)) {
		const target = el instanceof HTMLElement ? el.closest<HTMLElement>('[data-epic-id]') : null
		const id = parseInt(target?.dataset.epicId ?? '', 10)
		if (target && !Number.isNaN(id)) return {id, title: target.dataset.epicTitle ?? ''}
	}
	return null
}

/**
 * Lets the list and kanban views assign a task to an epic by dropping it on the epics panel.
 * Call handleTaskDropToEpic first in a drag end handler. It returns null right away when the task
 * was not dropped on the panel, so other drop handling keeps its timing; otherwise await the promise.
 */
export function useTaskDropToEpic() {
	const {draggedTask, setDraggedTask} = useTaskDragState()
	const assignToEpic = useAssignTaskToEpicMutation()

	function handleTaskDropToEpic(e: {originalEvent?: MouseEvent}): Promise<unknown> | null {
		const task = draggedTask.value
		if (!task || !e.originalEvent) return null

		const target = findEpicDropTarget(e.originalEvent.clientX, e.originalEvent.clientY)
		if (!target) return null

		setDraggedTask(null)
		if ((task.epic_id ?? 0) === target.id || task.is_epic) return Promise.resolve()

		// The mutation reports a failure itself.
		return assignToEpic.mutateAsync({taskId: task.id, epic: target.id > 0 ? target : null}).catch(() => undefined)
	}

	return {handleTaskDropToEpic}
}
