<template>
	<div
		v-cy="'kanban-swimlanes'"
		class="swimlanes"
		:style="{'--bucket-count': buckets.length}"
	>
		<div class="swimlanes__row swimlanes__bucket-row">
			<div
				v-for="bucket in buckets"
				:key="bucket.id"
				class="swimlanes__bucket-title"
			>
				<span>{{ bucket.title }}</span>
				<span class="swimlanes__count">{{ bucket.count }}</span>
				<BaseButton
					v-if="bucketHasMore(bucket)"
					class="swimlanes__more"
					@click="emit('loadMore', bucket.id)"
				>
					{{ $t('epic.group.loadMore') }}
				</BaseButton>
			</div>
		</div>

		<section
			v-for="lane in lanes"
			:key="lane.key"
			v-cy="'kanban-swimlane'"
			class="swimlanes__lane"
			:style="lane.epic ? getEpicChipStyles(lane.epic.hex_color) : undefined"
		>
			<EpicGroupHeader
				class="swimlanes__lane-header"
				:epic="lane.epic"
				:count="lane.tasks.length"
				:collapsed="collapsed.has(lane.key)"
				@toggle="collapsed.toggle(lane.key)"
			/>

			<div
				v-show="!collapsed.has(lane.key)"
				class="swimlanes__row"
			>
				<TaskDraggable
					v-for="bucket in buckets"
					:key="bucket.id"
					:model-value="cells[cellKey(lane.key, bucket.id)] ?? []"
					:group="{name: 'swimlane-tasks'}"
					:disabled="!canWrite"
					item-key="id"
					tag="ul"
					class="swimlanes__cell"
					:component-data="{'data-bucket-id': bucket.id, 'data-epic-group': lane.key}"
					v-bind="TASK_DRAG_OPTIONS"
					ghost-class="ghost"
					@update:modelValue="value => cells[cellKey(lane.key, bucket.id)] = value"
					@start="onDragStart"
					@end="onDragEnd"
				>
					<template #item="{element: task}">
						<li
							class="swimlanes__item"
							:data-task-id="task.id"
						>
							<KanbanCard
								class="kanban-card"
								:task="task"
								:project-id="projectId"
								:hide-epic="true"
							/>
						</li>
					</template>
				</TaskDraggable>
			</div>
		</section>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import KanbanCard from '@/components/tasks/partials/KanbanCard.vue'
import EpicGroupHeader from '@/components/epics/EpicGroupHeader.vue'
import {draggableFor} from '@/components/base/Draggable'

import {bucketHasMore, type BucketResponse} from '@/client/queries/kanban'
import type {TaskResponse} from '@/client/queries/tasks'
import {useAssignTaskToEpicMutation, useMoveTaskMutation, useUpdateTaskPositionMutation} from '@/client/queries/taskMutations'
import {useProjectEpics} from '@/composables/useProjectEpics'
import {useTaskDragState} from '@/composables/useTaskDragState'
import {useTaskDropToEpic} from '@/composables/useTaskDropToEpic'
import {useToggleSet} from '@/composables/useToggleSet'
import {calculateItemPosition} from '@/helpers/calculateItemPosition'
import {getEpicChipStyles} from '@/helpers/epicColor'
import {groupTasksByEpic, type EpicGroup} from '@/helpers/epicGroups'
import {NO_EPIC} from '@/constants/epics'
import {TASK_DRAG_OPTIONS} from '@/constants/taskDrag'

const props = defineProps<{
	buckets: BucketResponse[],
	projectId: number,
	viewId: number,
	canWrite: boolean,
}>()

const emit = defineEmits<{
	loadMore: [bucketId: number],
}>()

const TaskDraggable = draggableFor<TaskResponse>()
const {epics} = useProjectEpics(() => props.projectId)
const collapsed = useToggleSet<string>()

function cellKey(laneKey: string, bucketId: number) {
	return `${laneKey}:${bucketId}`
}

// Epic lanes in the panel's order, then the lane for tasks without an epic, which always takes drops.
const lanes = computed<EpicGroup[]>(() => {
	const groups = groupTasksByEpic(props.buckets.flatMap(bucket => bucket.tasks), epics.value)
	return groups.some(group => group.key === NO_EPIC)
		? groups
		: [...groups, {key: NO_EPIC, epic: null, tasks: []}]
})

function buildCells(): Record<string, TaskResponse[]> {
	return Object.fromEntries(props.buckets.flatMap(bucket =>
		groupTasksByEpic(bucket.tasks, epics.value).map(group => [cellKey(group.key, bucket.id), [...group.tasks]]),
	))
}

// Copies the draggables reorder until the server answers.
const cells = ref<Record<string, TaskResponse[]>>({})
const dragging = ref(false)
watch([() => props.buckets, epics], () => {
	if (!dragging.value) cells.value = buildCells()
}, {immediate: true, deep: true})

const {setDraggedTask} = useTaskDragState()
const {handleTaskDropToEpic} = useTaskDropToEpic()
const assignToEpic = useAssignTaskToEpicMutation()
const moveMutation = useMoveTaskMutation()
const positionMutation = useUpdateTaskPositionMutation()

function findTask(item: HTMLElement) {
	const id = Number(item.dataset.taskId)
	return props.buckets.flatMap(bucket => bucket.tasks).find(task => task.id === id)
}

function onDragStart(e: {item: HTMLElement}) {
	dragging.value = true
	const task = findTask(e.item)
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

		const task = findTask(e.item)
		if (!task) return
		const targetLane = e.to.dataset.epicGroup ?? NO_EPIC
		const targetBucketId = Number(e.to.dataset.bucketId)
		const list = cells.value[cellKey(targetLane, targetBucketId)] ?? []
		const index = list.findIndex(item => item.id === task.id)
		if (index === -1) return

		// Lanes, not the task's epic_id: a task whose epic is not loaded sits in the no-epic lane.
		if (targetLane !== e.from.dataset.epicGroup) {
			const epic = lanes.value.find(lane => lane.key === targetLane)?.epic ?? null
			// A failure throws into the catch below, which skips the position update.
			await assignToEpic.mutateAsync({taskId: task.id, epic})
		}
		if (targetBucketId !== Number(e.from.dataset.bucketId)) {
			const result = await moveMutation.mutateAsync({project: props.projectId, view: props.viewId, bucket: targetBucketId, task})
			if (result.bucket_id !== undefined && result.bucket_id !== targetBucketId) return
		}
		await positionMutation.mutateAsync({
			taskId: task.id,
			project_view_id: props.viewId,
			position: calculateItemPosition(list[index - 1]?.position ?? null, list[index + 1]?.position ?? null),
		})
	} catch { /* Mutations report their errors. */ } finally {
		dragging.value = false
		cells.value = buildCells()
	}
}
</script>

<style lang="scss" scoped>
.swimlanes {
	--swimlane-column: 18rem;

	display: flex;
	flex-direction: column;
	gap: .75rem;
	overflow-x: auto;
	padding-block-end: 1rem;
}

.swimlanes__row {
	display: grid;
	grid-template-columns: repeat(var(--bucket-count), var(--swimlane-column));
	gap: .75rem;
	align-items: stretch;
}

.swimlanes__bucket-row {
	position: sticky;
	inset-block-start: 0;
	z-index: 2;
	background: var(--site-background);
	padding-block: .25rem;
}

.swimlanes__bucket-title {
	display: flex;
	align-items: center;
	gap: .5rem;
	font-weight: 700;
	padding: .4rem .6rem;
	background: var(--grey-100);
	border-radius: $radius;
}

.swimlanes__count {
	font-size: .75rem;
	font-weight: 600;
	padding: 0 .4rem;
	border-radius: 999px;
	background: var(--white);
	color: var(--grey-600);
}

.swimlanes__more {
	margin-inline-start: auto;
	font-size: .75rem;
	font-weight: 400;
	color: var(--primary);
}

.swimlanes__lane {
	--epic-color: var(--grey-300);

	inline-size: max-content;
	min-inline-size: 100%;
	border-inline-start: 4px solid var(--epic-color);
	border-radius: $radius;
	background: color-mix(in srgb, var(--epic-color) 6%, transparent);
	padding: .25rem .5rem .5rem;
}

.swimlanes__lane-header {
	position: sticky;
	inset-inline-start: 0;
	inline-size: fit-content;
}

.swimlanes__cell {
	display: flex;
	flex-direction: column;
	gap: .5rem;
	min-block-size: 3rem;
	margin: 0;
	padding: .35rem;
	list-style: none;
	border-radius: $radius;
	background: var(--grey-100);
}

.swimlanes__item {
	list-style: none;
}

.ghost {
	opacity: .4;
}
</style>
