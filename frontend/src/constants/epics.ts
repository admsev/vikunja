// Group, lane and filter key for tasks without an epic. The API uses epic_id 0 for them.
export const NO_EPIC = 'none'

// Jira-like purple for epics without their own color.
export const DEFAULT_EPIC_COLOR = '#8f5bd9'

export const EPIC_STORAGE_KEYS = {
	panelOpen: 'vikunja-epics-panel',
	panelShowDone: 'vikunja-epics-panel-show-done',
	grouping: 'vikunja-epic-grouping',
} as const
