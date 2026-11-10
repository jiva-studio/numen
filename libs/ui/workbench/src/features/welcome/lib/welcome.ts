/**
 * What the screen a window opens on is drawn over.
 *
 * Both windows open on it: the editor while it holds no tab, and flashcards
 * while it has not been sent into a vault. The rows differ and the screen does
 * not.
 */
import type { Component } from 'vue'
import type { Keystroke } from '@numen/ui'

/** One entry point: its name, its icon, and the keystroke that triggers it. */
export interface WelcomeAction {
  readonly id: string
  readonly text: string
  readonly icon?: Component
  readonly keys?: Keystroke
}

/** One vault of the list, as the screen draws it. */
export interface VaultRow {
  readonly id: string
  readonly name: string
  readonly path: string
  readonly detail?: string
  readonly isWorking?: boolean
}

/** What the screen offers below the list, where a window offers anything. */
export interface FooterEntry {
  readonly text: string
  readonly detail?: string
  readonly icon?: Component
  readonly keys?: Keystroke
}
