import {describe, expect, it} from 'vitest'

import {NO_EPIC} from '@/constants/epics'
import {parseEpicFilter, regularTaskParams, withEpicFilter} from './useEpicFilter'

describe('parseEpicFilter', () => {
	it('reads epic ids and the no-epic marker', () => {
		expect(parseEpicFilter('12')).toBe(12)
		expect(parseEpicFilter(NO_EPIC)).toBe(NO_EPIC)
	})

	it('ignores anything else', () => {
		for (const raw of [undefined, '', '0', '-3', '1.5', 'abc']) {
			expect(parseEpicFilter(raw)).toBeNull()
		}
	})
})

describe('withEpicFilter', () => {
	it('leaves the filter alone without an epic filter', () => {
		expect(withEpicFilter('done = false', null)).toBe('done = false')
	})

	it('adds the epic condition to an existing filter', () => {
		expect(withEpicFilter('done = false', 12)).toBe('(done = false) && epic = 12')
		expect(withEpicFilter('', NO_EPIC)).toBe('epic = 0')
	})

	it('can keep the epic itself in the result', () => {
		expect(withEpicFilter('', 12, true)).toBe('(epic = 12 || id = 12)')
		expect(withEpicFilter('', NO_EPIC, true)).toBe('epic = 0')
	})
})

describe('regularTaskParams', () => {
	it('excludes epics in real projects', () => {
		expect(regularTaskParams(3, '', 12)).toEqual({filter: 'epic = 12', epics: 'exclude'})
	})

	it('keeps epics and ignores the epic filter in pseudo projects', () => {
		expect(regularTaskParams(-1, 'done = false', 12)).toEqual({filter: 'done = false'})
	})
})
