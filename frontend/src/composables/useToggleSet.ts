import {shallowRef} from 'vue'

/** A reactive set of keys that can be switched on and off, e.g. collapsed sections. */
export function useToggleSet<T>() {
	const keys = shallowRef(new Set<T>())

	function toggle(key: T) {
		const next = new Set(keys.value)
		if (!next.delete(key)) next.add(key)
		keys.value = next
	}

	return {
		has: (key: T) => keys.value.has(key),
		toggle,
	}
}
