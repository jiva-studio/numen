/**
 * The decorations the markdown is drawn with, one kept to a class, and the
 * names of the constructs each one belongs to.
 */
import { Decoration } from '@codemirror/view'

export const HEADING = /^(?:ATX|Setext)Heading([1-6])$/
export const BULLET = /^[-+*]$/

/** The marks that are drawn only while the selection is in what they belong to. */
export const SHY = new Set([
  'EmphasisMark',
  'StrikethroughMark',
  'LinkMark',
  'URL',
  'LinkTitle',
  'LinkLabel',
])

/** The class each stretch of marked-up text is drawn in. */
export const INLINE = new Map([
  ['InlineCode', 'cm-code-inline'],
  ['Emphasis', 'cm-em'],
  ['StrongEmphasis', 'cm-strong'],
  ['Strikethrough', 'cm-strike'],
  ['Link', 'cm-link'],
  ['Autolink', 'cm-link'],
])

/** What takes a mark out of the drawing. */
export const hidden = Decoration.replace({})

const lines = new Map<string, Decoration>()
const named = new Map<string, Decoration>()

/** The decoration that puts a class on a whole line. */
export const line = (name: string) => {
  const drawn = lines.get(name) ?? Decoration.line({ class: name })
  lines.set(name, drawn)
  return drawn
}

/** The decoration that puts a class on a stretch of text. */
export const mark = (name: string) => {
  const drawn = named.get(name) ?? Decoration.mark({ class: name })
  named.set(name, drawn)
  return drawn
}
