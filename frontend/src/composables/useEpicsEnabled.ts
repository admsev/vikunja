import {computed} from 'vue'

import {useAuthStore} from '@/stores/auth'

// Epics are opt in per user. Turning them off only hides the UI; tasks keep their epics.
export function useEpicsEnabled() {
	const authStore = useAuthStore()
	return computed(() => authStore.settings.frontend_settings.epics_enabled === true)
}
