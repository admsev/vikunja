import {describe, it, expect} from 'vitest'
import {flushPromises, mount} from '@vue/test-utils'
import {createI18n} from 'vue-i18n'
import {createRouter, createMemoryHistory} from 'vue-router'

import GanttChart from './GanttChart.vue'
import GanttTimelineHeader from './GanttTimelineHeader.vue'
import GanttRowBars from './GanttRowBars.vue'
import en from '@/i18n/lang/en.json'
import {i18n as globalI18n} from '@/i18n'
import {normalizeTask, type TaskResponse} from '@/client/queries/tasks'
import type {Task} from '@/client/generated'
import {DEFAULT_EPIC_COLOR} from '@/constants/epics'
import type {GanttFilters} from '@/views/project/helpers/useGanttFilters'

const i18n = createI18n({legacy: false, locale: 'en', messages: {en}})

// the dayjs locale sync reads the app-wide i18n instance, not the one installed on the wrapper
globalI18n.global.locale.value = 'en'
const router = createRouter({history: createMemoryHistory(), routes: [{path: '/', component: {template: '<div/>'}}]})

const FILTERS: GanttFilters = {
	projectId: 1,
	viewId: 1,
	dateFrom: '2026-08-01T00:00:00.000Z',
	dateTo: '2026-10-01T00:00:00.000Z',
	showTasksWithoutDates: false,
}

function mountChart(isLoading: boolean, tasks = new Map<number, TaskResponse>(), collapsedTaskIds = new Set<number>()) {
	return mount(GanttChart, {
		shallow: true,
		props: {
			isLoading,
			filters: FILTERS,
			tasks,
			collapsedTaskIds,
			defaultTaskStartDate: FILTERS.dateFrom,
			defaultTaskEndDate: FILTERS.dateTo,
		},
		global: {
			plugins: [i18n, router],
			// The rows render in the body's slot.
			stubs: {GanttChartBody: false, GanttChartPrimitive: false, GanttRow: false, GanttRowPrimitive: false},
		},
	})
}

describe('GanttChart.vue', () => {
	it('measures the day width once the chart replaces the loading state', async () => {
		const wrapper = mountChart(true)
		expect(wrapper.findComponent(GanttTimelineHeader).exists()).toBe(false)

		await wrapper.setProps({isLoading: false})

		expect(wrapper.findComponent(GanttTimelineHeader).props('dayWidthPixels')).toBeGreaterThan(0)
	})

	describe('epics', () => {
		const day = (n: number) => `2026-08-${String(n).padStart(2, '0')}T00:00:00.000Z`
		const task = (id: number, fields: Partial<Task>) => normalizeTask({id, title: `Task ${id}`, related_tasks: {}, ...fields})
		const tasks = new Map<number, TaskResponse>([
			[1, task(1, {is_epic: true})],
			[2, task(2, {epic_id: 1, start_date: day(5), end_date: day(9)})],
			[3, task(3, {start_date: day(6), end_date: day(8)})],
		])
		const rowIds = (wrapper: ReturnType<typeof mountChart>) =>
			wrapper.findAllComponents(GanttRowBars).map(rowBars => rowBars.props('bars')[0]?.id)

		it('hides the tasks of a collapsed epic and keeps its span', async () => {
			const wrapper = mountChart(false, tasks, new Set([1]))
			await flushPromises()

			expect(rowIds(wrapper)).toEqual(['1', '3'])
			const epicBar = wrapper.findAllComponents(GanttRowBars)[0].props('bars')[0]
			expect(epicBar.start.getDate()).toBe(new Date(day(5)).getDate())
		})

		it('draws an epic in the epic color', async () => {
			const wrapper = mountChart(false, tasks)
			await flushPromises()

			const bars = wrapper.findAllComponents(GanttRowBars).map(rowBars => rowBars.props('bars')[0])
			expect(bars[0].meta).toMatchObject({isEpic: true, color: DEFAULT_EPIC_COLOR})
			expect(bars[1].meta?.isEpic).toBeFalsy()
		})
	})
})
