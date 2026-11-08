import type {APIRequestContext, Page} from '@playwright/test'
import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {BucketFactory} from '../../factories/bucket'
import {TaskBucketFactory} from '../../factories/task_buckets'
import {createDefaultViews} from '../project/prepareProjects'
import {updateUserSettings} from '../../support/updateUserSettings'

const LIST_VIEW = 1
const GANTT_VIEW = 2
const TABLE_VIEW = 3
const KANBAN_VIEW = 4

// Project 1 with the default views, an epic "Launch plan" (#1), two of its tasks (#2, #3),
// a task without an epic (#4), and a second epic "Docs" (#5). Regular tasks sit in the first
// of two kanban buckets.
async function seedEpics() {
	const [project] = await ProjectFactory.create(1)
	await createDefaultViews(project.id)
	const buckets = await BucketFactory.create(2, {project_view_id: KANBAN_VIEW})
	const day = (n: number) => new Date(Date.UTC(2026, 2, n)).toISOString()
	const epic = (await TaskFactory.create(1, {id: 1, title: 'Launch plan', is_epic: true, project_id: project.id}))[0]
	const inEpic = [
		(await TaskFactory.create(1, {id: 2, title: 'Press release', epic_id: 1, start_date: day(2), end_date: day(5)}, false))[0],
		(await TaskFactory.create(1, {id: 3, title: 'Landing page', epic_id: 1, start_date: day(4), end_date: day(9)}, false))[0],
	]
	const loose = (await TaskFactory.create(1, {id: 4, title: 'Fix login bug'}, false))[0]
	const docs = (await TaskFactory.create(1, {id: 5, title: 'Docs', is_epic: true}, false))[0]
	for (const task of [...inEpic, loose]) {
		await TaskBucketFactory.create(1, {task_id: task.id, bucket_id: buckets[0].id, project_view_id: KANBAN_VIEW}, false)
	}
	return {project, buckets, epic, inEpic, loose, docs}
}

async function storedTask(apiContext: APIRequestContext, token: string, id: number) {
	const response = await apiContext.get(`tasks/${id}`, {headers: {Authorization: `Bearer ${token}`}})
	expect(response.ok()).toBeTruthy()
	return response.json()
}

async function openPanel(page: Page) {
	await page.getByTestId('epics-panel-toggle').click()
	await expect(page.getByTestId('epics-panel')).toBeVisible()
}

function panelEpic(page: Page, title: string) {
	return page.getByTestId('epics-panel-epic').filter({hasText: title})
}

test.describe('Epics', () => {
	// Epics are opt in per user.
	test.beforeEach(async ({apiContext, userToken}) => {
		await updateUserSettings(apiContext, userToken, {frontend_settings: {epics_enabled: true}})
	})

	test('creates an epic in the panel and narrows the list to it', async ({authenticatedPage: page}) => {
		await seedEpics()
		await page.goto(`/projects/1/${LIST_VIEW}`)
		await openPanel(page)

		await page.getByTestId('epics-panel-create').click()
		await page.getByTestId('epics-panel-create-input').fill('Onboarding')
		await page.getByTestId('epics-panel-create-input').press('Enter')
		await expect(panelEpic(page, 'Onboarding')).toBeVisible()

		await panelEpic(page, 'Launch plan').click()
		await expect(page).toHaveURL(/epic=1/)
		const titles = page.locator('.tasks .task-link')
		await expect(titles).toHaveText(['Press release', 'Landing page'])

		await page.reload()
		await expect(panelEpic(page, 'Onboarding')).toBeVisible()
		// Epics are managed in the panel, not listed among their tasks.
		await expect(titles).toHaveText(['Press release', 'Landing page'])
	})

	test('sets an epic on a task and shows it in the list and table', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {loose} = await seedEpics()
		await page.goto(`/tasks/${loose.id}`)

		await page.getByTestId('task-action-epic').click()
		const picker = page.getByTestId('epic-select').locator('input')
		await picker.fill('Launch')
		await page.getByTestId('epic-select').locator('.search-result-button').filter({hasText: 'Launch plan'}).click()
		await expect(page.locator('.subtitle').getByTestId('epic-chip')).toContainText('Launch plan')

		expect((await storedTask(apiContext, userToken, loose.id)).epic_id).toBe(1)
		await page.goto(`/projects/1/${TABLE_VIEW}`)
		await expect(page.locator('tr').filter({hasText: 'Fix login bug'}).getByTestId('epic-chip')).toContainText('Launch plan')
	})

	test('manages the tasks of an epic on its page', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {epic, loose} = await seedEpics()
		await page.goto(`/tasks/${epic.id}`)
		const rows = page.getByTestId('epic-task-row')
		await expect(rows).toHaveCount(2)

		await page.getByTestId('epic-tasks-create-input').fill('Write announcement')
		await page.getByTestId('epic-tasks-create-input').press('Enter')
		await expect(rows).toHaveCount(3)

		const addExisting = page.getByTestId('epic-tasks-add-existing').locator('input')
		await addExisting.fill('login')
		await page.getByTestId('epic-tasks-add-existing').locator('.search-result-button').filter({hasText: 'Fix login bug'}).click()
		await expect(rows).toHaveCount(4)
		expect((await storedTask(apiContext, userToken, loose.id)).epic_id).toBe(epic.id)

		await rows.filter({hasText: 'Press release'}).locator('.fancy-checkbox').click()
		await expect(page.getByTestId('epic-tasks')).toContainText('1/4')

		await rows.filter({hasText: 'Landing page'}).getByRole('button', {name: 'Remove from epic'}).click()
		await expect(rows).toHaveCount(3)
		await page.reload()
		await expect(rows).toHaveCount(3)
		expect((await storedTask(apiContext, userToken, 3)).epic_id).toBe(0)
	})

	test('moves a task to another epic by dragging it between list groups', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {loose} = await seedEpics()
		await page.goto(`/projects/1/${LIST_VIEW}`)
		await page.getByTestId('epic-group-toggle').click()

		const groups = page.getByTestId('epic-group')
		await expect(groups.filter({hasText: 'Launch plan'})).toContainText('Press release')
		const task = groups.filter({hasText: 'No epic'}).locator('.single-task').filter({hasText: loose.title})
		await task.dragTo(groups.filter({hasText: 'Launch plan'}).locator('.single-task').first())

		await expect(groups.filter({hasText: 'Launch plan'})).toContainText(loose.title)
		await expect.poll(async () => (await storedTask(apiContext, userToken, loose.id)).epic_id).toBe(1)
		await page.reload()
		await expect(groups.filter({hasText: 'Launch plan'})).toContainText(loose.title)
	})

	test('moves a card across swimlanes and columns on the board', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {loose, buckets} = await seedEpics()
		await page.goto(`/projects/1/${KANBAN_VIEW}`)
		await page.getByTestId('epic-group-toggle').click()

		const lanes = page.getByTestId('kanban-swimlane')
		await expect(lanes.filter({hasText: 'Launch plan'})).toContainText('Press release')
		const card = lanes.filter({hasText: 'No epic'}).locator('.kanban-card').filter({hasText: loose.title})
		const target = lanes.filter({hasText: 'Launch plan'}).locator(`ul[data-bucket-id="${buckets[1].id}"]`)
		await card.dragTo(target)

		await expect.poll(async () => (await storedTask(apiContext, userToken, loose.id)).epic_id).toBe(1)
		await expect.poll(async () => {
			const response = await apiContext.get(`tasks/${loose.id}?expand=buckets`, {headers: {Authorization: `Bearer ${userToken}`}})
			return (await response.json()).buckets?.map((bucket: {id: number}) => bucket.id)
		}).toEqual([buckets[1].id])
		await expect(target).toContainText(loose.title)
		await page.reload()
		await expect(lanes.filter({hasText: 'Launch plan'}).locator(`ul[data-bucket-id="${buckets[1].id}"]`)).toContainText(loose.title)
	})

	test('moves a task into an epic by dropping it on the panel', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {loose} = await seedEpics()
		await page.goto(`/projects/1/${LIST_VIEW}`)
		await openPanel(page)

		await page.locator('.tasks .single-task').filter({hasText: loose.title}).dragTo(panelEpic(page, 'Docs'))

		await expect(page.locator('.global-notification')).toContainText('Docs')
		await expect.poll(async () => (await storedTask(apiContext, userToken, loose.id)).epic_id).toBe(5)
	})

	test('nests the tasks of an epic below it on the gantt chart', async ({authenticatedPage: page}) => {
		await seedEpics()
		await page.goto(`/projects/1/${GANTT_VIEW}?dateFrom=2026-03-01&dateTo=2026-03-31`)

		const rows = page.locator('.gantt-row-content, [role="row"]')
		await expect(page.getByText('Launch plan').first()).toBeVisible()
		const titles = await rows.allInnerTexts()
		const epicRow = titles.findIndex(text => text.includes('Launch plan'))
		expect(epicRow).toBeGreaterThanOrEqual(0)
		expect(titles.findIndex(text => text.includes('Press release'))).toBeGreaterThan(epicRow)
	})

	test('collapses and expands an epic on the gantt chart', async ({authenticatedPage: page}) => {
		await seedEpics()
		await page.goto(`/projects/1/${GANTT_VIEW}?dateFrom=2026-03-01&dateTo=2026-03-31`)
		const row = (title: string) => page.locator('.gantt-row-content').filter({hasText: title})
		await expect(row('Press release')).toBeVisible()

		await page.getByRole('button', {name: 'Collapse group: Launch plan'}).click()
		await expect(row('Press release')).toHaveCount(0)
		await expect(row('Launch plan')).toBeVisible()

		await page.getByRole('button', {name: 'Expand group: Launch plan'}).click()
		await expect(row('Press release')).toBeVisible()
	})

	test('collapses all epics on the gantt chart at once', async ({authenticatedPage: page}) => {
		await seedEpics()
		await page.goto(`/projects/1/${GANTT_VIEW}?dateFrom=2026-03-01&dateTo=2026-03-31`)
		const row = (title: string) => page.locator('.gantt-row-content').filter({hasText: title})
		await expect(row('Landing page')).toBeVisible()

		await page.getByRole('button', {name: 'Collapse all epics'}).click()
		await expect(row('Launch plan')).toBeVisible()
		await expect(row('Press release')).toHaveCount(0)
		await expect(row('Landing page')).toHaveCount(0)

		await page.getByRole('button', {name: 'Expand all epics'}).click()
		await expect(row('Press release')).toBeVisible()
		await expect(row('Landing page')).toBeVisible()
	})

	test('draws epics in their color on the gantt chart', async ({authenticatedPage: page}) => {
		await seedEpics()
		await page.goto(`/projects/1/${GANTT_VIEW}?dateFrom=2026-03-01&dateTo=2026-03-31`)

		const epicBar = page.locator('.gantt-row-content').filter({hasText: 'Launch plan'}).locator('.gantt-bar rect').first()
		await expect(epicBar).toHaveAttribute('fill', '#8f5bd9')
		const taskBar = page.locator('.gantt-row-content').filter({hasText: 'Press release'}).locator('rect.gantt-bar')
		await expect(taskBar).not.toHaveAttribute('fill', '#8f5bd9')
	})

	test('keeps the epic when a task is duplicated', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {inEpic} = await seedEpics()
		await page.goto(`/tasks/${inEpic[0].id}`)

		await page.getByRole('button', {name: 'Duplicate'}).click()
		await expect(page).not.toHaveURL(new RegExp(`/tasks/${inEpic[0].id}$`))
		const copyId = Number(page.url().match(/tasks\/(\d+)/)?.[1])
		expect((await storedTask(apiContext, userToken, copyId)).epic_id).toBe(1)
	})

	test('keeps the tasks of an epic through trash and restore', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {epic, inEpic} = await seedEpics()
		await page.goto(`/tasks/${epic.id}`)

		await page.getByRole('button', {name: 'Delete'}).click()
		// Trashing an epic keeps its tasks attached.
		await expect.poll(async () => (await storedTask(apiContext, userToken, inEpic[0].id)).epic_id).toBe(epic.id)
		await page.locator('.global-notification').getByRole('button', {name: 'Undo'}).click()

		await page.goto(`/projects/1/${LIST_VIEW}`)
		await openPanel(page)
		await expect(panelEpic(page, 'Launch plan')).toContainText('0/2')
		expect((await storedTask(apiContext, userToken, inEpic[0].id)).epic_id).toBe(epic.id)
	})

	test('puts a quick-added task into the epic named with ~', async ({authenticatedPage: page}) => {
		await seedEpics()
		await page.goto(`/projects/1/${LIST_VIEW}`)

		await page.locator('.input[placeholder="Add a task…"]').fill('Write copy ~"Launch plan"')
		await page.locator('.input[placeholder="Add a task…"]').press('Enter')
		const created = page.locator('.tasks .single-task').filter({hasText: 'Write copy'})
		await expect(created).toBeVisible()
		await expect(created).not.toContainText('~')
		await expect(created.getByTestId('epic-chip')).toContainText('Launch plan')
	})
})

test.describe('Epics turned off', () => {
	test('hides epics until the user turns them on', async ({authenticatedPage: page}) => {
		await seedEpics()
		await page.goto(`/projects/1/${LIST_VIEW}`)
		await expect(page.locator('.tasks .single-task').filter({hasText: 'Press release'})).toBeVisible()
		await expect(page.getByTestId('epics-panel-toggle')).toHaveCount(0)
		await expect(page.getByTestId('epic-group-toggle')).toHaveCount(0)
		await expect(page.getByTestId('epic-chip')).toHaveCount(0)

		await page.goto('/user/settings/general')
		await page.getByLabel('Enable epics').check()
		await page.getByTestId('saveGeneralSettings').click()
		await expect(page.locator('.global-notification')).toContainText('Success')

		await page.goto(`/projects/1/${LIST_VIEW}`)
		await expect(page.getByTestId('epics-panel-toggle')).toBeVisible()
		await expect(page.locator('.tasks .single-task').filter({hasText: 'Press release'}).getByTestId('epic-chip')).toContainText('Launch plan')
	})
})
