/**
 * What a composer is, as plain values. No DOM, no clock, no measurement.
 */

export interface ComposerStateDescriptor {
  /** Whether there is a button to press at all. */
  readonly acts: boolean
  /** What stands at the end of the field. */
  readonly shows: 'send' | 'writing'
}

/**
 * Every state the composer can be in, declared once. What stands at the end
 * of the field and whether it can be pressed both read from here.
 */
export const COMPOSER_STATES = {
  empty: { acts: false, shows: 'send' },
  ready: { acts: true, shows: 'send' },
  writing: { acts: false, shows: 'writing' },
} as const satisfies Record<string, ComposerStateDescriptor>

export type ComposerState = keyof typeof COMPOSER_STATES

/**
 * Which state a composer holding this text is in.
 *
 * Writing outranks the text: the end of the field belongs to the answer on
 * its way until it arrives.
 */
export const composerState = (text: string, working: boolean): ComposerState => {
  if (working) return 'writing'
  return said(text) ? 'ready' : 'empty'
}

/** What is left after the whitespace, which is what would be sent. */
export const said = (text: string): string => text.trim()

/** What the key that was pressed means. */
export type KeyIntent = 'submit' | 'newline' | 'pass'

/**
 * A key, read as an intent.
 *
 * Enter sends and Shift+Enter breaks the line. While a text input method is
 * composing, Enter is choosing a candidate and belongs to the input method.
 * Composition is read from the event and from the key both: a browser
 * mid-composition reports the key as 229.
 */
export const keyIntent = (event: {
  readonly key: string
  readonly shiftKey: boolean
  readonly isComposing?: boolean
  readonly keyCode?: number
}): KeyIntent => {
  if (event.key !== 'Enter') return 'pass'
  if (event.isComposing || event.keyCode === 229) return 'pass'
  return event.shiftKey ? 'newline' : 'submit'
}
