import {getTextColor} from '@/helpers/color/getTextColor'
import {DEFAULT_EPIC_COLOR} from '@/constants/epics'

/** The color an epic is drawn in: its own, or the default epic color. */
export function getEpicColor(hexColor?: string): string {
	if (!hexColor) return DEFAULT_EPIC_COLOR
	return hexColor.startsWith('#') ? hexColor : `#${hexColor}`
}

/** CSS variables for elements drawn in an epic's color. */
export function getEpicChipStyles(hexColor?: string) {
	const color = getEpicColor(hexColor)
	return {
		'--epic-color': color,
		'--epic-text-color': getTextColor(color),
	}
}
