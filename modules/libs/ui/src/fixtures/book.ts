/**
 * A document of a book, marked up the way a reader is handed one: every run of
 * text carrying the byte offset it begins at in the book's one text stream.
 *
 * The offsets are counted here the way they are counted where they come from —
 * in bytes, through `TextEncoder` — so a story written in Devanagari or in
 * Cyrillic carries the numbers such a book really carries. Storybook's
 * furniture; it ships to nobody.
 */
import { bytesIn, type Span } from '@/book/spread'

/** One run of a document. */
export interface Run {
  /** The element the run is set in. */
  readonly tag: string
  /** The run's own text, whose bytes are what the offsets count. */
  readonly text?: string
  /** Markup standing in the run that the text stream carries nothing of. */
  readonly inside?: string
}

/** One document of a book, and where it stands in the book's text. */
export interface Chapter {
  readonly markup: string
  readonly span: Span
}

const escaped = (text: string): string =>
  text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

/**
 * The runs written out as one document, each carrying where it begins. A run
 * with no text of its own — a picture, a table — carries the offset the run
 * after it begins at and adds nothing to the stream.
 */
export function chapterOf(runs: readonly Run[], begins = 0): Chapter {
  let at = begins
  const written: string[] = []
  for (const run of runs) {
    const text = run.text ?? ''
    written.push(
      `<${run.tag} data-offset="${at}">${escaped(text)}${run.inside ?? ''}</${run.tag}>`,
    )
    at += bytesIn(text)
  }
  return { markup: written.join('\n'), span: { begins, ends: at } }
}

/** A page of Sanskrit in Devanagari, its transliteration and its rendering. */
export const VERSES: readonly Run[] = [
  { tag: 'h2', text: 'Слово о свете' },
  {
    tag: 'p',
    text: 'सत्यं ज्ञानमनन्तं ब्रह्म — the syllables are three bytes apiece, and the offsets a book carries count bytes and not letters.',
  },
  {
    tag: 'p',
    text: 'Каждая буква кириллицы — два байта, и место, на котором остановился читатель, названо байтом, а не страницей.',
  },
  {
    tag: 'p',
    text: 'A reader sent to a passage is sent to an offset, and the offset stands wherever the columns happen to have put it.',
  },
]

/**
 * Verse as a book converted from plain text carries it: one pre element holding
 * the lines it was written on. Long enough to run over several spreads, so a
 * document that stopped at the first is one that stopped.
 */
export const VERSE: string = Array.from(
  { length: 48 },
  (_, index) =>
    `  ${index + 1}. Then spake the Blessed One, and thus he said:\n` +
    '     "Know thou the Self, that neither slays nor dies,\n' +
    '      unborn, unending, ancient of the worlds."',
).join('\n')

/** Enough prose to run over several spreads at any size the text is set at. */
export const PROSE: readonly Run[] = Array.from({ length: 24 }, (_, index) => ({
  tag: 'p',
  text:
    `${index + 1}. ` +
    'The text is set in columns as wide as the reading area, the whole of it ' +
    'held at one height and moved sideways by one spread at a time. Nothing ' +
    'scrolls: a page is turned, and the place a person is reading is the ' +
    'offset of the first run standing in front of them.',
}))
