<template>
	<section
		v-cy="'epic-tasks'"
		class="epic-tasks"
		:style="getEpicChipStyles(epic.hex_color)"
	>
		<header class="epic-tasks__header">
			<h2 class="task-section-title">
				<span class="icon is-grey">
					<Icon icon="bolt" />
				</span>
				{{ $t('epic.tasks.title') }}
			</h2>
			<EpicProgress
				class="epic-tasks__progress"
				size="normal"
				:progress="progress"
			/>
		</header>

		<p
			v-if="!isPending && tasks.length === 0"
			class="epic-tasks__empty"
		>
			{{ $t('epic.tasks.empty') }}
		</p>

		<ul class="epic-tasks__list">
			<li
				v-for="child in tasks"
				:key="child.id"
				v-cy="'epic-task-row'"
				class="epic-tasks__row"
				:class="{'is-done': child.done}"
			>
				<FancyCheckbox
					:model-value="child.done"
					:disabled="!canWrite"
					:aria-label="$t('task.detail.markAsDone', {task: child.title})"
					@update:modelValue="done => toggleDone(child, done)"
				/>
				<span class="epic-tasks__identifier">{{ child.identifier }}</span>
				<RouterLink
					:to="{name: 'task.detail', params: {id: child.id}}"
					class="epic-tasks__title"
				>
					{{ child.title }}
				</RouterLink>
				<span
					v-if="child.due_date && +new Date(child.due_date) > 0"
					class="epic-tasks__due"
				>
					{{ formatDisplayDate(child.due_date) }}
				</span>
				<AssigneeList
					v-if="child.assignees.length > 0"
					:assignees="child.assignees"
					:avatar-size="20"
					:inline="true"
				/>
				<BaseButton
					v-if="canWrite"
					v-tooltip="$t('epic.removeTask')"
					class="epic-tasks__remove"
					:aria-label="$t('epic.removeTask')"
					@click="assignToEpic.mutate({taskId: child.id, epic: null})"
				>
					<Icon icon="times" />
				</BaseButton>
			</li>
		</ul>

		<div
			v-if="canWrite"
			class="epic-tasks__add"
		>
			<form @submit.prevent="createChild">
				<input
					v-model="newTaskTitle"
					v-cy="'epic-tasks-create-input'"
					class="input"
					:placeholder="$t('epic.tasks.createPlaceholder')"
					:aria-disabled="createTask.isPending.value || undefined"
				>
			</form>
			<Multiselect
				v-cy="'epic-tasks-add-existing'"
				:placeholder="$t('epic.tasks.addPlaceholder')"
				:search-results="candidates"
				label="title"
				:model-value="null"
				:search-delay="CANDIDATE_SEARCH_DELAY"
				:close-after-select="true"
				:aria-label="$t('epic.tasks.addExisting')"
				@search="query => candidateQuery = query"
				@select="candidate => assignToEpic.mutate({taskId: candidate.id, epic})"
			>
				<template #searchResult="{option}">
					<span class="has-text-grey">{{ option.identifier }}</span>
					{{ option.title }}
				</template>
			</Multiselect>
		</div>
	</section>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import BaseButton from '@/components/base/BaseButton.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import Icon from '@/components/misc/Icon'
import Multiselect from '@/components/input/Multiselect.vue'
import AssigneeList from '@/components/tasks/partials/AssigneeList.vue'
import EpicProgress from '@/components/epics/EpicProgress.vue'

import {tasksQuery, type TaskResponse} from '@/client/queries/tasks'
import {PICKER_PAGE_SIZE} from '@/client/queries/pagination'
import {
	useAssignTaskToEpicMutation,
	useCreateTaskMutation,
	useUpdateTaskMutation,
} from '@/client/queries/taskMutations'
import {withEpicFilter} from '@/composables/useEpicFilter'
import {useEpicTasks} from '@/composables/useEpicTasks'
import {getEpicChipStyles} from '@/helpers/epicColor'
import {formatDisplayDate} from '@/helpers/time/formatDate'
import {NO_EPIC} from '@/constants/epics'

const props = defineProps<{
	epic: {id: number, title: string, project_id: number, hex_color?: string},
	canWrite: boolean,
}>()

// Milliseconds of typing pause before the candidates are searched.
const CANDIDATE_SEARCH_DELAY = 200

const {tasks, isPending, progress} = useEpicTasks(() => props.epic)

const updateTask = useUpdateTaskMutation()
function toggleDone(task: TaskResponse, done: boolean) {
	updateTask.mutate({...task, done})
}

const assignToEpic = useAssignTaskToEpicMutation()

const createTask = useCreateTaskMutation()
const newTaskTitle = ref('')
async function createChild() {
	const typed = newTaskTitle.value
	const title = typed.trim()
	if (title === '' || createTask.isPending.value) return
	try {
		await createTask.mutateAsync({title, project_id: props.epic.project_id, epic_id: props.epic.id})
	} catch {
		return
	}
	// Keep whatever was typed while the task was being created.
	if (newTaskTitle.value === typed) newTaskTitle.value = ''
}

// Only open tasks without an epic are offered; moving a task between epics happens on the task itself.
const candidateQuery = ref('')
const candidatesQuery = useQuery(computed(() => ({
	...tasksQuery({
		project: props.epic.project_id,
		params: {
			q: candidateQuery.value,
			filter: withEpicFilter('done = false', NO_EPIC),
			epics: 'exclude',
			per_page: PICKER_PAGE_SIZE,
		},
	}),
	enabled: props.canWrite,
})))
const candidates = computed(() => candidatesQuery.data.value?.items ?? [])
</script>

<style lang="scss" scoped>
.epic-tasks {
	display: flex;
	flex-direction: column;
	gap: .5rem;
	margin-block-end: 1.5rem;
}

.epic-tasks__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;

	.task-section-title {
		margin-block-end: 0;
	}
}

.epic-tasks__progress {
	flex: 0 1 16rem;
}

.epic-tasks__empty {
	color: var(--grey-500);
	font-size: .875rem;
}

.epic-tasks__list {
	list-style: none;
	margin: 0;
	padding: 0;
	border: 1px solid var(--grey-200);
	border-radius: $radius;

	&:empty {
		display: none;
	}
}

.epic-tasks__row {
	display: flex;
	align-items: center;
	gap: .5rem;
	padding: .4rem .6rem;

	&:not(:last-child) {
		border-block-end: 1px solid var(--grey-100);
	}

	&.is-done .epic-tasks__title {
		text-decoration: line-through;
		color: var(--grey-500);
	}
}

.epic-tasks__identifier {
	font-size: .75rem;
	color: var(--grey-500);
	white-space: nowrap;
}

.epic-tasks__title {
	flex: 1;
	min-inline-size: 0;
	color: var(--text);
}

.epic-tasks__due {
	font-size: .8rem;
	color: var(--grey-500);
	white-space: nowrap;
}

.epic-tasks__remove {
	color: var(--grey-400);
	padding: .15rem .3rem;

	&:hover {
		color: var(--danger);
	}
}

.epic-tasks__add {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: .5rem;

	@media screen and (max-width: $tablet) {
		grid-template-columns: 1fr;
	}
}
</style>
