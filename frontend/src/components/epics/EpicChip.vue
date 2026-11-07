<script setup lang="ts">
import {computed} from 'vue'

import Icon from '@/components/misc/Icon'
import {useProjectEpics} from '@/composables/useProjectEpics'
import {useEpicFilter} from '@/composables/useEpicFilter'
import {getEpicChipStyles} from '@/helpers/epicColor'

const props = withDefaults(defineProps<{
	epicId: number | undefined,
	projectId: number,
	// Clicking narrows the current project view to the epic.
	filterOnClick?: boolean,
}>(), {
	filterOnClick: true,
})

const {getEpicById} = useProjectEpics(() => props.projectId)
const {toggleEpicFilter} = useEpicFilter()

const epic = computed(() => getEpicById(props.epicId))

function onClick(e: MouseEvent) {
	if (!props.filterOnClick || !epic.value) return
	e.preventDefault()
	e.stopPropagation()
	toggleEpicFilter(epic.value.id)
}
</script>

<template>
	<span
		v-if="epic"
		v-cy="'epic-chip'"
		class="epic-chip"
		:class="{'is-done': epic.done, 'is-clickable': filterOnClick}"
		:style="getEpicChipStyles(epic.hex_color)"
		:title="`${$t('task.attributes.epic')}: ${epic.title}`"
		@click="onClick"
	>
		<Icon
			icon="bolt"
			class="epic-chip__icon"
		/>
		<span class="epic-chip__title">{{ epic.title }}</span>
	</span>
</template>

<style lang="scss" scoped>
.epic-chip {
	display: inline-flex;
	align-items: center;
	gap: .25rem;
	max-inline-size: 14rem;
	padding: .1rem .5rem;
	border-radius: 999px;
	border: 1px solid var(--epic-color);
	background: color-mix(in srgb, var(--epic-color) 14%, transparent);
	color: var(--text);
	font-size: .75rem;
	font-weight: 600;
	line-height: 1.4;
	vertical-align: middle;
	white-space: nowrap;

	&.is-clickable {
		cursor: pointer;

		&:hover {
			background: color-mix(in srgb, var(--epic-color) 26%, transparent);
		}
	}

	&.is-done .epic-chip__title {
		text-decoration: line-through;
		opacity: .7;
	}
}

.epic-chip__icon {
	color: var(--epic-color);
	font-size: .7rem;
}

.epic-chip__title {
	overflow: hidden;
	text-overflow: ellipsis;
}
</style>
