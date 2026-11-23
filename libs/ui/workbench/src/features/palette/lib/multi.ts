/** The rows of a step that asks for several items, as the palette draws them. */
import type { PaletteGroup, PaletteItem, PaletteItemState } from './item'

export interface PaletteMultiItem {
  readonly id: string
  readonly title: string
  readonly state: PaletteItemState
  readonly color?: string
}

/**
 * A step that asks for several items from a list. The ids of `recent` come most recent first.
 * `getCreateTitle` and `getCreateColor` name and paint the row that creates an item from the text typed.
 */
export interface PaletteMultiProps {
  readonly title: string
  readonly placeholder: string
  readonly items: readonly PaletteMultiItem[]
  readonly recent: readonly string[]
  readonly getCreateTitle: (text: string) => string
  readonly getCreateColor?: (text: string) => string
}

/** The id of the row that creates an item. */
export const CREATE_ID = '\u0000create'

const TOGGLE = { id: 'toggle', text: 'Toggle' }
const CREATE = { id: 'create', text: 'Create' }

const isOn = (item: PaletteMultiItem): boolean => item.state === 'set' || item.state === 'mixed'

const compareByTitle = (one: PaletteMultiItem, other: PaletteMultiItem): number =>
  one.title.localeCompare(other.title)

export const orderMultiItems = (
  items: readonly PaletteMultiItem[],
  recent: readonly string[],
): readonly PaletteMultiItem[] => {
  const on = items.filter(isOn).sort(compareByTitle)
  const off = items.filter((item) => !isOn(item))
  const recents = recent.flatMap((id) => off.find((item) => item.id === id) ?? [])
  const rest = off.filter((item) => !recents.includes(item)).sort(compareByTitle)
  return [...on, ...recents, ...rest]
}

const filterMultiItems = (
  items: readonly PaletteMultiItem[],
  query: string,
): readonly PaletteMultiItem[] => {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean)
  return items.filter((item) => words.every((word) => item.title.toLowerCase().includes(word)))
}

const toRow = (item: PaletteMultiItem): PaletteItem => ({
  id: item.id,
  title: item.title,
  state: item.state,
  ...(item.color === undefined ? {} : { color: item.color }),
  actions: [TOGGLE],
})

/**
 * The one group the step shows for the text typed: `items`, the items of the step in the order to show,
 * narrowed to the text, and ahead of them the row that creates an item once the text is not empty and no
 * item has it as its title.
 */
export const getMultiGroup = (
  multi: PaletteMultiProps,
  items: readonly PaletteMultiItem[],
  query: string,
): PaletteGroup => {
  const shown = filterMultiItems(items, query)
  const text = query.trim()
  const isNamed = multi.items.some((item) => item.title.trim().toLowerCase() === text.toLowerCase())
  const color = text === '' ? undefined : multi.getCreateColor?.(text)
  const create: PaletteItem = {
    id: CREATE_ID,
    title: text === '' ? '' : multi.getCreateTitle(text),
    ...(color === undefined ? {} : { color }),
    actions: [CREATE],
  }
  return {
    id: 'multi',
    title: multi.title,
    items: text === '' || isNamed ? shown.map(toRow) : [create, ...shown.map(toRow)],
  }
}
