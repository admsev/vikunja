<template>
	<XButton
		v-cy="'epic-group-toggle'"
		:variant="modelValue ? 'primary' : 'secondary'"
		icon="layer-group"
		:aria-pressed="modelValue"
		@click="emit('update:modelValue', !modelValue)"
	>
		{{ label }}
	</XButton>
</template>

<script setup lang="ts">
defineProps<{
	modelValue: boolean,
	label: string,
}>()

const emit = defineEmits<{
	'update:modelValue': [value: boolean],
}>()
</script>
