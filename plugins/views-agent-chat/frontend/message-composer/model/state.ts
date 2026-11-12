/**
 * What a composer is, as plain values. No DOM, no clock, no measurement.
 */

/** What the disc at the end of the field does when it is pressed. */
export type ComposerAction = 'send' | 'stop'

export interface ComposerStateDescriptor {
  readonly canAct: boolean
  readonly action: ComposerAction
}

/** The states of a composer. The glyph and label of the disc and whether it can be pressed read from here. */
export const COMPOSER_STATES = {
  empty: { canAct: false, action: 'send' },
  ready: { canAct: true, action: 'send' },
  pending: { canAct: true, action: 'stop' },
} as const satisfies Record<string, ComposerStateDescriptor>

export type ComposerState = keyof typeof COMPOSER_STATES

/**
 * The state of a composer holding this text.
 *
 * A pending answer outranks the text: the disc stops the answer until it settles.
 */
export const getComposerState = (text: string, isAnswerPending: boolean): ComposerState => {
  if (isAnswerPending) return 'pending'
  return getMessage(text) ? 'ready' : 'empty'
}

/** What is left after the whitespace, which is what would be sent. */
export const getMessage = (text: string): string => text.trim()

/** What the key that was pressed means. */
export type KeyIntent = 'submit' | 'newline' | 'pass'

/**
 * A key, read as an intent.
 * Enter sends and Shift+Enter breaks the line. While a text input method is composing, Enter belongs to it.
 * Composition is read from the event and from the key both: a browser mid-composition reports the key as 229.
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
