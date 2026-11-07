import {computed, toValue, type MaybeRefOrGetter} from 'vue'
import {useLocalStorage} from '@vueuse/core'

import {EPIC_STORAGE_KEYS} from '@/constants/epics'

// Whether a project view shows its tasks grouped by epic, remembered per view.
export function useEpicGrouping(projectId: MaybeRefOrGetter<number>, viewId: MaybeRefOrGetter<number>) {
	const stored = useLocalStorage<Record<string, boolean>>(EPIC_STORAGE_KEYS.grouping, {})
	const key = computed(() => `${toValue(projectId)}:${toValue(viewId)}`)

	return computed({
		// Epics are per project, so pseudo projects can't be grouped.
		get: () => toValue(projectId) > 0 && (stored.value[key.value] ?? false),
		set: (value: boolean) => { stored.value = {...stored.value, [key.value]: value} },
	})
}
