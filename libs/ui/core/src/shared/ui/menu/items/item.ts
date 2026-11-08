/**
 * What a menu is, as plain values. No DOM, no measurement, no clock.
 */
import type { Component } from 'vue'
import { getAdjacentPlace } from '@/shared/lib/dom/place'
import { stepTo } from '@/shared/lib/dom/roving'
import type { Position, Size } from '@/shared/lib/math/geometry'

/**
 * One selectable entry. The identifier is opaque: the menu cannot inspect what
 * it addresses, and returns it unchanged.
 */
export interface MenuItem {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly isDisabled?: boolean
  readonly group?: string
  readonly children?: readonly MenuItem[]
  readonly component?: Component
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
  readonly at: Position
  readonly size: Size
  readonly viewport: Size
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
  readonly hasInitialFocus: boolean
}

const MENU_TRIGGERS = {
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
