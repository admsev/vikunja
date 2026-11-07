<template>
	<aside
		v-cy="'epics-panel'"
		class="epics-panel"
		:class="{'is-dragging': draggedTask !== null}"
		:aria-label="$t('epic.panel.title')"
	>
		<header class="epics-panel__header">
			<h2
				class="epics-panel__title"
				:style="getEpicChipStyles()"
			>
				<Icon icon="bolt" />
				{{ $t('epic.panel.title') }}
				<span
					v-if="epics.length > 0"
					class="epics-panel__count"
				>{{ epics.length }}</span>
			</h2>
			<BaseButton
				v-tooltip="$t('epic.panel.toggleHint')"
				class="epics-panel__close"
				:aria-label="$t('epic.panel.toggleHint')"
				@click="emit('close')"
			>
				<Icon icon="times" />
			</BaseButton>
		</header>

		<ul class="epics-panel__list">
			<li>
				<BaseButton
					class="epics-panel__row epics-panel__row--plain"
					:class="{'is-active': epicFilter === null}"
					@click="epicFilter = null"
				>
					<Icon
						icon="layer-group"
						class="epics-panel__row-icon"
					/>
					{{ $t('epic.panel.all') }}
				</BaseButton>
			</li>
			<li>
				<BaseButton
					class="epics-panel__row epics-panel__row--plain"
					:class="{'is-active': epicFilter === 'none'}"
					data-epic-id="0"
					@click="toggleEpicFilter('none')"
				>
					<Icon
						icon="minus"
						class="epics-panel__row-icon"
					/>
					{{ $t('epic.panel.none') }}
				</BaseButton>
			</li>
		</ul>

		<p
			v-if="!isPending && epics.length === 0"
			class="epics-panel__empty"
		>
			{{ $t('epic.panel.empty') }}
		</p>

		<ul
			v-else
			class="epics-panel__list epics-panel__epics"
		>
			<li
				v-for="epic in visibleEpics"
				:key="epic.id"
			>
				<div
					v-cy="'epics-panel-epic'"
					class="epics-panel__row epics-panel__epic"
					:class="{'is-active': epicFilter === epic.id, 'is-done': epic.done}"
					:style="getEpicChipStyles(epic.hex_color)"
					:data-epic-id="epic.id"
					:data-epic-title="epic.title"
					role="button"
					tabindex="0"
					@click="toggleEpicFilter(epic.id)"
					@keyup.enter="toggleEpicFilter(epic.id)"
				>
					<div class="epics-panel__epic-heading">
						<span class="epics-panel__epic-title">{{ epic.title }}</span>
						<RouterLink
							:to="{name: 'task.detail', params: {id: epic.id}}"
							class="epics-panel__open"
							:aria-label="epic.title"
							@click.stop.prevent="openEpic(epic.id)"
						>
							<Icon icon="arrow-up-right-from-square" />
						</RouterLink>
					</div>
					<EpicProgress
						class="epics-panel__progress"
						:progress="epic.epic_progress"
					/>
				</div>
			</li>
		</ul>

		<form
			v-if="canWrite && creating"
			class="epics-panel__create"
			@submit.prevent="submitNewEpic"
		>
			<input
				v-model="newEpicTitle"
				v-focus
				v-cy="'epics-panel-create-input'"
				class="input is-small"
				:placeholder="$t('epic.createPlaceholder')"
				:aria-disabled="isCreating || undefined"
				@keyup.esc="creating = false"
				@blur="newEpicTitle === '' && (creating = false)"
			>
		</form>
		<BaseButton
			v-else-if="canWrite"
			v-cy="'epics-panel-create'"
			class="epics-panel__row epics-panel__row--plain epics-panel__create-button"
			@click="creating = true"
		>
			<Icon
				icon="plus"
				class="epics-panel__row-icon"
			/>
			{{ $t('epic.create') }}
		</BaseButton>

		<footer class="epics-panel__footer">
			<FancyCheckbox
				v-if="doneCount > 0"
				v-model="showDone"
			>
				{{ $t('epic.panel.showDone') }} ({{ doneCount }})
			</FancyCheckbox>
			<p
				v-if="canWrite && epics.length > 0"
				class="epics-panel__hint"
			>
				{{ $t('epic.panel.dropHint') }}
			</p>
		</footer>
	</aside>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useRouter} from 'vue-router'
import {useLocalStorage} from '@vueuse/core'

import BaseButton from '@/components/base/BaseButton.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import Icon from '@/components/misc/Icon'
import EpicProgress from '@/components/epics/EpicProgress.vue'

import {useProjectEpics} from '@/composables/useProjectEpics'
import {useEpicFilter} from '@/composables/useEpicFilter'
import {useTaskDragState} from '@/composables/useTaskDragState'
import {useCreateEpicMutation} from '@/client/queries/taskMutations'
import {getEpicChipStyles} from '@/helpers/epicColor'
import {EPIC_STORAGE_KEYS} from '@/constants/epics'

const props = defineProps<{
	projectId: number,
	canWrite: boolean,
}>()

const emit = defineEmits<{
	close: [],
}>()

const router = useRouter()

const {epics, isPending} = useProjectEpics(() => props.projectId)
const {epicFilter, toggleEpicFilter} = useEpicFilter()
const {draggedTask} = useTaskDragState()

const showDone = useLocalStorage(EPIC_STORAGE_KEYS.panelShowDone, false)
const doneCount = computed(() => epics.value.filter(epic => epic.done).length)
const visibleEpics = computed(() => epics.value.filter(epic =>
	showDone.value || !epic.done || epic.id === epicFilter.value))

function openEpic(id: number) {
	router.push({
		name: 'task.detail',
		params: {id},
		state: {backdropView: router.currentRoute.value.fullPath},
	})
}

const creating = ref(false)
const newEpicTitle = ref('')
const createEpic = useCreateEpicMutation()
const isCreating = createEpic.isPending

async function submitNewEpic() {
	const typed = newEpicTitle.value
	const title = typed.trim()
	if (title === '' || isCreating.value) return
	try {
		await createEpic.mutateAsync({title, project_id: props.projectId, is_epic: true})
	} catch {
		return
	}
	// Keep whatever was typed while the epic was being created.
	if (newEpicTitle.value === typed) newEpicTitle.value = ''
}
</script>

<style lang="scss" scoped>
.epics-panel {
	display: flex;
	flex-direction: column;
	gap: .5rem;
	padding: .75rem;
	background: var(--white);
	border-radius: $radius;
	box-shadow: var(--shadow-sm);
	align-self: start;
	position: sticky;
	inset-block-start: 1rem;
	max-block-size: calc(100vh - 2rem);
	overflow-y: auto;
}

.epics-panel__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
}

.epics-panel__title {
	display: flex;
	align-items: center;
	gap: .4rem;
	margin: 0;
	font-size: 1rem;
	font-weight: 700;

	.icon, svg {
		color: var(--epic-color);
	}
}

.epics-panel__count {
	font-size: .75rem;
	font-weight: 600;
	padding: 0 .4rem;
	border-radius: 999px;
	background: var(--grey-100);
	color: var(--grey-600);
}

.epics-panel__close {
	color: var(--grey-400);
	padding: .25rem;

	&:hover {
		color: var(--grey-700);
	}
}

.epics-panel__list {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: .25rem;
}

.epics-panel__row {
	inline-size: 100%;
	text-align: start;
	border-radius: $radius;
	padding: .4rem .5rem;
	cursor: pointer;
	transition: background-color 100ms;
	border: 2px solid transparent;

	&:hover {
		background: var(--grey-100);
	}

	&.is-active {
		background: var(--grey-100);
		border-color: var(--primary);
	}
}

.epics-panel__row--plain {
	display: flex;
	align-items: center;
	gap: .5rem;
	font-size: .875rem;
	color: var(--grey-700);
}

.epics-panel__row-icon {
	inline-size: 1rem;
	color: var(--grey-400);
}

.epics-panel__epic {
	display: flex;
	flex-direction: column;
	gap: .3rem;
	border-inline-start: 4px solid var(--epic-color);
	background: color-mix(in srgb, var(--epic-color) 7%, transparent);

	&:hover {
		background: color-mix(in srgb, var(--epic-color) 15%, transparent);
	}

	&.is-active {
		border-color: var(--epic-color);
		background: color-mix(in srgb, var(--epic-color) 18%, transparent);
	}

	&.is-done .epics-panel__epic-title {
		text-decoration: line-through;
		color: var(--grey-500);
	}
}

.epics-panel__epic-heading {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: .5rem;
}

.epics-panel__epic-title {
	font-weight: 600;
	font-size: .875rem;
	overflow-wrap: anywhere;
}

.epics-panel__open {
	color: var(--grey-400);
	font-size: .75rem;
	opacity: 0;
	transition: opacity 100ms;

	.epics-panel__epic:hover &,
	.epics-panel__epic:focus-within & {
		opacity: 1;
	}

	&:hover {
		color: var(--primary);
	}
}

.epics-panel__progress {
	display: flex;
}

.epics-panel__empty,
.epics-panel__hint {
	font-size: .8rem;
	color: var(--grey-500);
	margin: 0;
}

.epics-panel__footer {
	display: flex;
	flex-direction: column;
	gap: .25rem;
	font-size: .8rem;
}

// While a task is dragged, every drop target is outlined.
.epics-panel.is-dragging [data-epic-id] {
	border: 2px dashed var(--epic-color, var(--grey-400));
}
</style>
