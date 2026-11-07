<template>
	<div
		v-cy="'epic-groups'"
		class="epic-groups"
	>
		<section
			v-for="group in groups"
			:key="group.key"
			v-cy="'epic-group'"
			class="epic-group"
			:style="group.epic ? getEpicChipStyles(group.epic.hex_color) : undefined"
		>
			<EpicGroupHeader
				:epic="group.epic"
				:count="group.tasks.length"
				:collapsed="collapsed.has(group.key)"
				@toggle="collapsed.toggle(group.key)"
			/>

			<draggable
				v-show="!collapsed.has(group.key)"
				:model-value="groupTasks[group.key] ?? []"
				:group="{name: 'epic-tasks'}"
				:disabled="!canDrag"
				item-key="id"
				tag="ul"
				:component-data="{class: 'tasks epic-group__tasks', 'data-epic-group': group.key}"
				v-bind="TASK_DRAG_OPTIONS"
				ghost-class="task-ghost"
				@update:modelValue="value => groupTasks[group.key] = value"
				@start="onDragStart"
				@end="onDragEnd"
			>
				<template #item="{element: task}">
					<SingleTaskInProject
						:show-list-color="false"
						:can-mark-as-done="canMarkAsDone"
						:the-task="task"
						:all-tasks="allTasks"
						:hide-epic="true"
						@taskUpdated="emit('taskUpdated')"
					/>
				</template>
			</draggable>
		</section>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'

import SingleTaskInProject from '@/components/tasks/partials/SingleTaskInProject.vue'
import EpicGroupHeader from '@/components/epics/EpicGroupHeader.vue'
import {draggableFor} from '@/components/base/Draggable'

import {useProjectEpics} from '@/composables/useProjectEpics'
import {useTaskDragState} from '@/composables/useTaskDragState'
import {useTaskDropToEpic} from '@/composables/useTaskDropToEpic'
import {useToggleSet} from '@/composables/useToggleSet'
import {useAssignTaskToEpicMutation, useUpdateTaskPositionMutation} from '@/client/queries/taskMutations'
import type {TaskResponse} from '@/client/queries/tasks'
import {calculateItemPosition} from '@/helpers/calculateItemPosition'
import {getEpicChipStyles} from '@/helpers/epicColor'
import {groupTasksByEpic} from '@/helpers/epicGroups'
import {TASK_DRAG_OPTIONS} from '@/constants/taskDrag'

const props = defineProps<{
	tasks: TaskResponse[],
	allTasks: TaskResponse[],
	projectId: number,
	viewId: number,
	canDrag: boolean,
	// Whether a drop also stores the new position; only meaningful while the list is sorted by it.
	reorderable: boolean,
	canMarkAsDone: boolean,
}>()

const emit = defineEmits<{
	taskUpdated: [],
}>()


const draggable = draggableFor<TaskResponse>()
const {epics} = useProjectEpics(() => props.projectId)
const groups = computed(() => groupTasksByEpic(props.tasks, epics.value))
const collapsed = useToggleSet<string>()

// Copies the draggables reorder until the server answers.
const groupTasks = ref<Record<string, TaskResponse[]>>({})
const dragging = ref(false)
function resetGroupTasks() {
	groupTasks.value = Object.fromEntries(groups.value.map(group => [group.key, [...group.tasks]]))
}
watch(groups, () => {
	if (!dragging.value) resetGroupTasks()
}, {immediate: true})

const {setDraggedTask} = useTaskDragState()
const {handleTaskDropToEpic} = useTaskDropToEpic()
const assignToEpic = useAssignTaskToEpicMutation()
const positionMutation = useUpdateTaskPositionMutation()

function taskIdOf(item: HTMLElement) {
	return Number(item.dataset.taskId)
}

function onDragStart(e: {item: HTMLElement}) {
	dragging.value = true
	const task = props.tasks.find(item => item.id === taskIdOf(e.item))
	if (task) setDraggedTask(task)
}

async function onDragEnd(e: {originalEvent?: MouseEvent, from: HTMLElement, to: HTMLElement, item: HTMLElement}) {
	try {
		const epicDrop = handleTaskDropToEpic(e)
		if (epicDrop) {
			await epicDrop
			return
		}
		setDraggedTask(null)

		const taskId = taskIdOf(e.item)
		const targetKey = e.to.dataset.epicGroup ?? ''
		const list = groupTasks.value[targetKey] ?? []
		const index = list.findIndex(task => task.id === taskId)
		if (index === -1) return

		if (e.from !== e.to) {
			const epic = groups.value.find(group => group.key === targetKey)?.epic ?? null
			// A failure throws into the catch below, which skips the position update.
			await assignToEpic.mutateAsync({taskId: taskId, epic})
		}
		if (!props.reorderable) return

		await positionMutation.mutateAsync({
			taskId,
			project_view_id: props.viewId,
			position: calculateItemPosition(list[index - 1]?.position ?? null, list[index + 1]?.position ?? null),
		})
	} catch { /* Mutations report their errors. */ } finally {
		dragging.value = false
		resetGroupTasks()
	}
}
</script>

<style lang="scss" scoped>
.epic-groups {
	display: flex;
	flex-direction: column;
	gap: .75rem;
	padding: .5rem;
}

.epic-group {
	--epic-color: var(--grey-300);

	border-inline-start: 4px solid var(--epic-color);
	border-radius: $radius;
	background: color-mix(in srgb, var(--epic-color) 5%, transparent);
}

.epic-group__tasks {
	margin: 0;
	padding: 0 .25rem .25rem;
	min-block-size: 2rem;
}
</style>
