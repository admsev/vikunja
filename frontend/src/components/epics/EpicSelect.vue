<template>
	<div class="epic-select">
		<Multiselect
			v-cy="'epic-select'"
			class="control is-expanded"
			:placeholder="$t('epic.select.placeholder')"
			:search-results="foundEpics"
			label="title"
			:model-value="selected"
			:creatable="canCreate"
			:create-placeholder="$t('epic.create')"
			:show-empty="true"
			:close-after-select="true"
			:disabled="disabled"
			:aria-label="$t('task.attributes.epic')"
			@search="value => query = value"
			@select="epic => emit('change', epic)"
			@create="createAndChoose"
		>
			<template #searchResult="{option}">
				<span
					class="epic-select__option"
					:style="getEpicChipStyles(option.hex_color)"
				>
					<Icon
						icon="bolt"
						class="epic-select__icon"
					/>
					<span :class="{'has-text-grey': option.done}">{{ option.title }}</span>
					<span class="epic-select__identifier">{{ option.identifier }}</span>
				</span>
			</template>
		</Multiselect>
		<BaseButton
			v-if="modelValue && !disabled"
			v-cy="'epic-select-remove'"
			class="epic-select__remove"
			@click="emit('change', null)"
		>
			{{ $t('epic.removeTask') }}
		</BaseButton>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'
import Multiselect from '@/components/input/Multiselect.vue'

import {useProjectEpics} from '@/composables/useProjectEpics'
import {useCreateEpicMutation, type EpicRef} from '@/client/queries/taskMutations'
import type {EpicResponse} from '@/client/queries/epics'
import {getEpicChipStyles} from '@/helpers/epicColor'

const props = withDefaults(defineProps<{
	// The id of the selected epic, 0 for none.
	modelValue: number,
	projectId: number,
	disabled?: boolean,
	canCreate?: boolean,
}>(), {
	disabled: false,
	canCreate: true,
})

const emit = defineEmits<{
	// The chosen epic, or null to take the task out of its epic.
	change: [epic: EpicRef | null],
}>()

const {epics, getEpicById} = useProjectEpics(() => props.projectId)
const createEpic = useCreateEpicMutation()

const selected = computed(() => getEpicById(props.modelValue) ?? null)

const query = ref('')
const foundEpics = computed<EpicResponse[]>(() => {
	const needle = query.value.trim().toLowerCase()
	return epics.value
		.filter(epic => epic.id !== props.modelValue)
		.filter(epic => needle === '' || epic.title.toLowerCase().includes(needle))
		// Done epics only show up when searched for.
		.filter(epic => !epic.done || needle !== '')
})

async function createAndChoose(title: string) {
	const trimmed = title.trim()
	if (trimmed === '') return
	let created
	try {
		created = await createEpic.mutateAsync({title: trimmed, project_id: props.projectId, is_epic: true})
	} catch {
		return
	}
	if (created.id !== undefined) emit('change', {id: created.id, title: trimmed})
}
</script>

<style lang="scss" scoped>
.epic-select {
	display: flex;
	flex-direction: column;
	gap: .25rem;
}

.epic-select__option {
	display: inline-flex;
	align-items: center;
	gap: .4rem;
}

.epic-select__icon {
	color: var(--epic-color);
}

.epic-select__identifier {
	font-size: .75rem;
	color: var(--grey-500);
}

.epic-select__remove {
	align-self: flex-start;
	font-size: .8rem;
	color: var(--grey-500);

	&:hover {
		color: var(--danger);
	}
}
</style>
