<template>
	<header
		class="epic-group-header"
		:class="{'is-collapsed': collapsed, 'is-no-epic': !epic}"
	>
		<BaseButton
			class="epic-group-header__toggle"
			:aria-expanded="!collapsed"
			@click="emit('toggle')"
		>
			<Icon
				icon="chevron-down"
				class="epic-group-header__chevron"
			/>
			<Icon
				v-if="epic"
				icon="bolt"
				class="epic-group-header__icon"
			/>
			<span class="epic-group-header__title">{{ epic ? epic.title : $t('epic.group.noEpic') }}</span>
			<span class="epic-group-header__count">{{ count }}</span>
		</BaseButton>
		<template v-if="epic">
			<EpicProgress :progress="epic.epic_progress" />
			<RouterLink
				class="epic-group-header__open"
				:to="{name: 'task.detail', params: {id: epic.id}}"
				:aria-label="epic.title"
			>
				<Icon icon="arrow-up-right-from-square" />
			</RouterLink>
		</template>
	</header>
</template>

<script setup lang="ts">
import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'
import EpicProgress from '@/components/epics/EpicProgress.vue'

import type {EpicResponse} from '@/client/queries/epics'

defineProps<{
	epic: EpicResponse | null,
	count: number,
	collapsed: boolean,
}>()

const emit = defineEmits<{
	toggle: [],
}>()
</script>

<style lang="scss" scoped>
.epic-group-header {
	display: flex;
	align-items: center;
	gap: .75rem;
	padding: .4rem .6rem;
}

.epic-group-header__toggle {
	display: inline-flex;
	align-items: center;
	gap: .4rem;
	min-inline-size: 0;
	font-weight: 700;
	text-align: start;
}

.epic-group-header__chevron {
	font-size: .7rem;
	color: var(--grey-500);
	transition: transform 150ms;

	.is-collapsed & {
		transform: rotate(-90deg);
	}
}

.epic-group-header__icon {
	color: var(--epic-color);
}

.epic-group-header__title {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;

	.is-no-epic & {
		color: var(--grey-600);
	}
}

.epic-group-header__count {
	font-size: .75rem;
	font-weight: 600;
	padding: 0 .4rem;
	border-radius: 999px;
	background: var(--grey-100);
	color: var(--grey-600);
}

.epic-group-header__open {
	color: var(--grey-400);
	font-size: .75rem;

	&:hover {
		color: var(--primary);
	}
}
</style>
