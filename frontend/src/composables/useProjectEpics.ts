import {computed, toValue, type MaybeRefOrGetter} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import {projectEpicsQuery, type EpicResponse} from '@/client/queries/epics'

// Every epic chip looks its epic up here; one index per fetched list instead of one per chip.
const indexes = new WeakMap<EpicResponse[], Map<number, EpicResponse>>()

function indexById(epics: EpicResponse[]): Map<number, EpicResponse> {
	let index = indexes.get(epics)
	if (!index) {
		index = new Map(epics.map(epic => [epic.id, epic]))
		indexes.set(epics, index)
	}
	return index
}

const NO_EPICS: EpicResponse[] = []

export function useProjectEpics(projectId: MaybeRefOrGetter<number>) {
	const query = useQuery(computed(() => projectEpicsQuery(toValue(projectId))))
	const epics = computed(() => query.data.value ?? NO_EPICS)
	const epicsById = computed(() => indexById(epics.value))

	return {
		epics,
		epicsById,
		isPending: query.isPending,
		getEpicById: (id: number | undefined) => id ? epicsById.value.get(id) : undefined,
	}
}
