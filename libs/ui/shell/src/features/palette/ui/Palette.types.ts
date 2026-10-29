import type { PaletteKeys } from '@numen/ui'
import type { ActionWords } from '../lib/actions'
import type { PaletteGroup } from '../lib/item'

export interface PaletteProps {
  /**
   * The groups, in the order they are offered. A group holding nothing is
   * is working or has something to say in place of items.
   */
  groups?: readonly PaletteGroup[]
  /** Whether it is drawn at all. */
  isOpen?: boolean
  /** The words standing in for what has not been typed. */
  placeholder?: string
  /**
   * Which step of a flow the field is on, in a few words, drawn beside it.
   * Nothing is drawn without it.
   */
  crumb?: string
  /**
   * Which step the caller is on. The identifier is opaque; changing it returns
   * the keyboard focus to the field with the current text selected.
   */
  step?: string
  /**
   * The item the keyboard relies on as a step opens: for a list of values,
   * the value in force. A list of no value, and a value with no item to
   * stand on, open on the first item there is.
   */
  opensOn?: string
  /** Where the keyboard goes back to once it closes. */
  from?: HTMLElement | null
  /** Where it is drawn. The end of the document by default. */
  to?: string | HTMLElement
  /** What it is announced as. */
  name?: string
  /** The words the action panel is drawn with. */
  actionWords?: ActionWords
  /** The keystroke that opens the action panel, as this machine reports it. */
  actionKey?: PaletteKeys
}
