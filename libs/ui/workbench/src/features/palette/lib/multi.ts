/** The rows of a step that asks for several items, as the palette draws them. */
import type { PaletteGroup, PaletteItem, PaletteItemState } from './item'

interface PaletteMultiItem {
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

const byTitle = (one: PaletteMultiItem, other: PaletteMultiItem): number =>
  one.title.localeCompare(other.title)

/** The items the set and the mixed first by title, then the recent ones in the order given, then the rest by title. */
const orderMultiItems = (
  items: readonly PaletteMultiItem[],
  recent: readonly string[],
): readonly PaletteMultiItem[] => {
  const on = items.filter(isOn).sort(byTitle)
  const off = items.filter((item) => !isOn(item))
  const recents = recent.flatMap((id) => off.find((item) => item.id === id) ?? [])
  const rest = off.filter((item) => !recents.includes(item)).sort(byTitle)
  return [...on, ...recents, ...rest]
}

/** The items whose title holds every word of the query, in any case. */
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
 * The one group the step shows for the text typed: the items in order, and ahead of them the row that
 * creates an item once the text is not empty and no item has it as its title.
 */
export const getMultiGroup = (multi: PaletteMultiProps, query: string): PaletteGroup => {
  const items = filterMultiItems(orderMultiItems(multi.items, multi.recent), query)
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
    items: text === '' || isNamed ? items.map(toRow) : [create, ...items.map(toRow)],
  }
}
