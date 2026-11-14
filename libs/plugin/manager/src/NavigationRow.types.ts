/**
 * What a navigation row draws. The accessible name is `label`, or `name` where none is given. A tile
 * (`isTile`) has no chevron; a static row (`isStatic`) is no button, opens nothing and has no chevron.
 */
export interface NavigationRowProps {
  readonly name: string
  readonly detail: string
  readonly category: string
  readonly label?: string
  readonly isDimmed?: boolean
  readonly isTile?: boolean
  readonly isStatic?: boolean
}
