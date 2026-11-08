<template>
	<ProjectWrapper
		class="project-gantt"
		:is-loading-project="isLoadingProject"
		:project-id
		:view-id
	>
		<template #default>
			<Card :has-content="false">
				<div class="gantt-options">
					<FormField
						id="range"
						:label="$t('misc.dateRange')"
					>
						<template #default="{ id }">
							<DateRangeInput
								:id="id"
								v-model="dateRange"
								:placeholder="$t('misc.dateRange')"
							/>
						</template>
					</FormField>
					<div
						v-if="!hasDefaultFilters"
						class="field"
					>
						<span
							class="label"
							aria-hidden="true"
						>Reset</span>
						<div class="control">
							<XButton @click="setDefaultFilters">
								Reset
							</XButton>
						</div>
					</div>
					<div class="gantt-options__toggles">
						<XButton
							v-if="epicIds.length > 0"
							variant="secondary"
							:wrap="false"
							:icon="allEpicsCollapsed ? 'chevron-down' : 'chevron-up'"
							:aria-pressed="allEpicsCollapsed"
							@click="toggleAllEpics"
						>
							{{ allEpicsCollapsed ? $t('project.gantt.expandEpics') : $t('project.gantt.collapseEpics') }}
						</XButton>
						<FancyCheckbox
							v-model="filters.showTasksWithoutDates"
							is-block
						>
							{{ $t('task.show.noDates') }}
						</FancyCheckbox>
					</div>
				</div>
			</Card>

			<div class="gantt-chart-container">
				<Card
					:has-content="false"
					:padding="false"
					class="has-overflow"
				>
					<GanttChart
						v-model:collapsed-task-ids="collapsedTaskIds"
						:filters="filters"
						:tasks="tasks"
						:is-loading="isLoading"
						:default-task-start-date="defaultTaskStartDate"
						:default-task-end-date="defaultTaskEndDate"
						@update:task="updateTask"
					/>
					<TaskForm
						v-if="canWrite"
						@createTask="addGanttTask"
					/>
				</Card>
			</div>
		</template>
	</ProjectWrapper>
</template>

<script setup lang="ts">
import {computed, ref, toRefs} from 'vue'
import type {RouteLocationNormalized} from 'vue-router'

import {useCurrentProject} from '@/composables/useCurrentProject'

import DateRangeInput from '@/components/input/DateRangeInput.vue'
import ProjectWrapper from '@/components/project/ProjectWrapper.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import FormField from '@/components/input/FormField.vue'

import GanttChart from '@/components/gantt/GanttChart.vue'
import {useGanttFilters} from '../../../views/project/helpers/useGanttFilters'
import {PERMISSIONS} from '@/constants/permissions'

import type {DateISO} from '@/types/DateISO'
import type {Task as ITask} from '@/client/generated'

const props = defineProps<{
	isLoadingProject: boolean,
	projectId: number,
	route: RouteLocationNormalized
	viewId: number
}>()


const {currentProject} = useCurrentProject()
const canWrite = computed(() =>
	typeof currentProject.value?.max_permission === 'number' &&
	currentProject.value.max_permission > PERMISSIONS.READ,
)

const {route, projectId, viewId} = toRefs(props)
const {
	filters,
	hasDefaultFilters,
	setDefaultFilters,
	tasks,
	isLoading,
	addTask,
	updateTask,
} = useGanttFilters(route, projectId, viewId)

const collapsedTaskIds = ref(new Set<number>())
// Empty while epics are turned off, which hides the toggle.
const epicIds = computed(() => [...tasks.value.values()].filter(task => task.is_epic).map(task => task.id))
const allEpicsCollapsed = computed(() => epicIds.value.length > 0 && epicIds.value.every(id => collapsedTaskIds.value.has(id)))

function toggleAllEpics() {
	const next = new Set(collapsedTaskIds.value)
	for (const id of epicIds.value) {
		if (allEpicsCollapsed.value) next.delete(id)
		else next.add(id)
	}
	collapsedTaskIds.value = next
}

const DEFAULT_DATE_RANGE_DAYS = 7

const today = new Date()
const defaultTaskStartDate: DateISO = new Date(today.setHours(0, 0, 0, 0)).toISOString()
const defaultTaskEndDate: DateISO = new Date(new Date(
	today.getFullYear(),
	today.getMonth(),
	today.getDate() + DEFAULT_DATE_RANGE_DAYS,
).setHours(23, 59, 0, 0)).toISOString()

async function addGanttTask(title: ITask['title']) {
	return await addTask({
		title,
		project_id: filters.value.projectId,
		start_date: defaultTaskStartDate,
		end_date: defaultTaskEndDate,
	})
}

const dateRange = computed({
	get: () => ({
		start: new Date(filters.value.dateFrom),
		end: new Date(filters.value.dateTo),
	}),
	set({start, end}: {start: Date, end: Date}) {
		Object.assign(filters.value, {dateFrom: start.toISOString(), dateTo: end.toISOString()})
	},
})
</script>

<style lang="scss" scoped>
.gantt-chart-container {
	padding-block-end: 1rem;
	position: relative;
	z-index: 0;
}

.gantt-options {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-block-end: 1rem;

	@media screen and (max-width: $tablet) {
		flex-direction: column;
	}
}

.gantt-options__toggles {
	display: flex;
	align-items: center;
	gap: 1rem;
	white-space: nowrap;
}

:global(.link-share-view:not(.has-background)) .gantt-options {
	border: none;
	box-shadow: none;

	.card-content {
		padding: .5rem;
	}
}

.field {
	margin-block-end: 0;
	inline-size: 33%;

	&:not(:last-child) {
		padding-inline-end: .5rem;
	}

	@media screen and (max-width: $tablet) {
		inline-size: 100%;
		max-inline-size: 100%;
		margin-block-start: .5rem;
		padding-inline-end: 0 !important;
	}

	&, .input {
		font-size: .8rem;
	}

	.select,
	.select select {
		block-size: auto;
		inline-size: 100%;
		font-size: .8rem;
	}

	.label {
		font-size: .9rem;
	}
}
</style>
