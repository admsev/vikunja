import {getTextColor} from '@/helpers/color/getTextColor'
import {DEFAULT_EPIC_COLOR} from '@/constants/epics'

/** CSS variables for elements drawn in an epic's color; without one, the default epic color. */
export function getEpicChipStyles(hexColor?: string) {
	const color = !hexColor
		? DEFAULT_EPIC_COLOR
		: hexColor.startsWith('#') ? hexColor : `#${hexColor}`
	return {
		'--epic-color': color,
		'--epic-text-color': getTextColor(color),
	}
}
