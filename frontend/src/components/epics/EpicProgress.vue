<template>
	<span
		class="epic-progress"
		:class="`is-${size}`"
	>
		<span
			class="epic-progress__bar"
			role="progressbar"
			:aria-valuenow="percent"
			aria-valuemin="0"
			aria-valuemax="100"
			:aria-label="$t('epic.progress', progress)"
		>
			<span :style="{inlineSize: `${percent}%`}" />
		</span>
		<span class="epic-progress__text">{{ progress.done }}/{{ progress.total }}</span>
	</span>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import {epicProgressPercent, type EpicProgress} from '@/client/queries/epics'

// Drawn in the surrounding --epic-color.
const props = withDefaults(defineProps<{
	progress: EpicProgress,
	size?: 'small' | 'normal',
}>(), {
	size: 'small',
})

const percent = computed(() => epicProgressPercent(props.progress))
</script>

<style lang="scss" scoped>
.epic-progress {
	display: inline-flex;
	align-items: center;
	gap: .5rem;
	font-size: .7rem;
	color: var(--grey-500);
	font-variant-numeric: tabular-nums;
}

.epic-progress__bar {
	flex: 1;
	min-inline-size: 4rem;
	block-size: .35rem;
	border-radius: 999px;
	background: var(--grey-200);
	overflow: hidden;

	.is-normal & {
		block-size: .5rem;
	}

	> span {
		display: block;
		block-size: 100%;
		background: var(--epic-color);
		border-radius: 999px;
		transition: inline-size 200ms;
	}
}
</style>
