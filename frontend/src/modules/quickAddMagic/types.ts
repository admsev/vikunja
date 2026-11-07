import type {IRepeatAfter} from '@/types/IRepeatAfter'

export interface repeatParsedResult {
	textWithoutMatched: string,
	repeats: IRepeatAfter | null,
}

export interface ParsedTaskText {
	text: string,
	date: Date | null,
	labels: string[],
	project: string | null,
	priority: number | null,
	assignees: string[],
	// Resolved against the project's epics later; the text keeps it until then.
	epic: string | null,
	repeats: IRepeatAfter | null,
}

export interface Prefixes {
	label: string,
	project: string,
	priority: string,
	assignee: string,
	epic: string,
}
