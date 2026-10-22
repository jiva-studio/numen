/**
 * What a menu is, as plain values. No DOM, no measurement, no clock.
 */
import { getPlaceBeside } from '@/shared/lib/place'
import { stepTo } from '@/shared/lib/roving'
import type { Position, Size } from '@/shared/lib/geometry'

/**
 * One thing that can be chosen. The identifier is opaque: the menu has no way
 * to ask what it addresses, and hands it back as given.
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
}

/** One item as it is drawn: what it says, and the rule standing above it. */
export interface GroupedItem extends MenuItem {
  /** It begins a group, and a rule stands between it and what is above. */
  readonly isRule: boolean
}

/**
 * The items in the order they were given, each saying whether a rule stands
 * above it. The first item begins the menu, and nothing is drawn above it.
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
  /** Where it was asked for. */
  readonly at: Position
  /** How big it turned out to be. */
  readonly size: Size
  /** The area it is placed in. */
  readonly viewport: Size
  /** Kept clear of that area's edges, so nothing sits flush against them. */
  readonly margin: number
}

/**
 * Where a menu of this size, asked for at this point, is drawn, in the
 * coordinates the point arrived in.
 *
 * The point is a span of no width, touching the menu: the menu runs on from it
 * and folds back over it at an edge. Wider than the area it is placed in, it
 * sits at the near edge and scrolls.
 */
export const placeMenu = ({ at, size, viewport, margin }: MenuPlacement): Position => ({
  x: getPlaceBeside({
    from: at.x,
    to: at.x,
    size: size.width,
    room: viewport.width,
    margin,
    gap: 0,
  }),
  y: getPlaceBeside({
    from: at.y,
    to: at.y,
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

/** What opened a menu: a hand, or the keyboard. */
export type MenuTrigger = keyof typeof MENU_TRIGGERS

export const MENU_TRIGGERS_ALL = Object.keys(MENU_TRIGGERS) as readonly MenuTrigger[]

/**
 * Where the keyboard is as a menu opens. Opened by hand it is on no item, and
 * the first step down from there lands on the first one. Opened by the
 * keyboard it lands on the item in force, and on the first where the menu
 * holds none.
 */
export const getInitialIndex = (
  trigger: MenuTrigger,
  items: readonly MenuItem[],
  current: string | null = null,
): number => {
  if (!MENU_TRIGGERS[trigger].hasInitialFocus) return -1
  const at = items.findIndex((item) => item.id === current && !item.isDisabled)
  return stepTo(items, { from: Math.max(at, 0) - 1, by: 1 }, canLandOn)
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
