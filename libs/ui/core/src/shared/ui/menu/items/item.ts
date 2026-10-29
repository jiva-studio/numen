/**
 * What a menu is, as plain values. No DOM, no measurement, no clock.
 */
import { getAdjacentPlace } from '@/shared/lib/dom/place'
import { stepTo } from '@/shared/lib/dom/roving'
import type { Position, Size } from '@/shared/lib/math/geometry'

/**
 * One selectable entry. The identifier is opaque: the menu cannot inspect what
 * it addresses, and returns it unchanged.
 */
export interface MenuItem {
  readonly id: string
  /** What is written on it. */
  readonly text: string
  /** A second line under the words, in the small print. */
  readonly detail?: string
  /** Drawn and announced, and not choosable. */
  readonly isDisabled?: boolean
  /**
   * The group it belongs to. Items of one group stand together, and a rule is
   * drawn where one group gives way to the next. A menu whose items name no
   * group is one group and carries no rules.
   */
  readonly group?: string
  /** Submenu items that expand from this row. */
  readonly children?: readonly MenuItem[]
}

/** One item as rendered: its label, and the separator above it. */
export interface GroupedItem extends MenuItem {
  readonly isRule: boolean
}

/**
 * The items in their given order, each flagged with whether a separator precedes
 * it. The first item begins the menu and has no separator above it.
 */
export const groupItems = (items: readonly MenuItem[]): readonly GroupedItem[] => {
  let above: MenuItem | undefined
  return items.map((item) => {
    const isRule = above !== undefined && item.group !== above.group
    above = item
    return { ...item, isRule }
  })
}

/** What placing a menu needs to know. */
export interface MenuPlacement {
  /** The point where the menu was requested. */
  readonly at: Position
  /** How big it turned out to be. */
  readonly size: Size
  /** The area it is placed in. */
  readonly viewport: Size
  /** Clearance from that area's edges. */
  readonly margin: number
}

/**
 * Position of a menu of this size, requested at this point, in the coordinates
 * of the point.
 *
 * The point is a span of zero width touching the menu: the menu extends from
 * it, is placed at the near edge and scrolls.
 */
export const placeMenu = ({ at: point, size, viewport, margin }: MenuPlacement): Position => ({
  x: getAdjacentPlace({
    from: point.x,
    to: point.x,
    size: size.width,
    room: viewport.width,
    margin,
    gap: 0,
  }),
  y: getAdjacentPlace({
    from: point.y,
    to: point.y,
    size: size.height,
    room: viewport.height,
    margin,
    gap: 0,
  }),
})

/**
 * How a menu came to be open, declared once. Where the keyboard is as it
 * appears is derived from this table.
 */
export interface MenuTriggerDescriptor {
  /** Whether the keyboard lands on an item as the menu appears. */
  readonly hasInitialFocus: boolean
}

export const MENU_TRIGGERS = {
  pointer: { hasInitialFocus: false },
  keyboard: { hasInitialFocus: true },
} as const satisfies Record<string, MenuTriggerDescriptor>

/** What opened a menu: a pointer, or the keyboard. */
export type MenuTrigger = keyof typeof MENU_TRIGGERS

export const MENU_TRIGGERS_ALL = Object.keys(MENU_TRIGGERS) as readonly MenuTrigger[]

/**
 * Initial keyboard focus when a menu opens. Opened by pointer, no item is
 * focused and the first step down lands on the first item. Opened by keyboard,
 * focus lands on the current item, or on the first item when the menu has none.
 */
export const getInitialIndex = (
  trigger: MenuTrigger,
  items: readonly MenuItem[],
  current: string | null = null,
): number => {
  if (!MENU_TRIGGERS[trigger].hasInitialFocus) return -1
  const pos = items.findIndex((item) => item.id === current && !item.isDisabled)
  return stepTo(items, { from: Math.max(pos, 0) - 1, by: 1 }, canLandOn)
}

/** Whether the keyboard may land on the item. */
export const canLandOn = (item: MenuItem): boolean => !item.isDisabled

/**
 * The items a search words name. Every word has to be found in an item's text,
 * detail or group, in any case; no words name every item.
 */
export const filterItems = (items: readonly MenuItem[], query: string): readonly MenuItem[] => {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean)
  if (words.length === 0) return items
  return items.filter((item) => {
    const haystack = [item.text, item.detail ?? '', item.group ?? ''].join('\n').toLowerCase()
    return words.every((word) => haystack.includes(word))
  })
}
