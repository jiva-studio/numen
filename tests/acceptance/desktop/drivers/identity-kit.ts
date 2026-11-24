/** The recovery kit of an identity as a PDF of sheets of words: reading its pages and planning what a scenario types or changes. */
import zlib from 'node:zlib'

export const SHEETS_IN_KIT = 3
export const WORDS_ON_SHEET = 33
const MISTAKE_AT = 5
const OTHER_ACCOUNTS: readonly (readonly string[])[] = [
  'letter advice cage absurd amount doctor acoustic avoid letter advice cage above'.split(' '),
  'legal winner thank year wave sausage worth useful legal winner thank yellow'.split(' '),
]

/** The numbers of the sheets a step names: "2" is the sheet 2, "1 and 3" are the sheets 1 and 3 in that order. */
export const readSheetNumbers = (text: string): number[] => {
  const parts = text.split(/\s+and\s+/)
  const numbers = parts.map((part) => (/^\d+$/.test(part.trim()) ? Number(part) : NaN))
  if (numbers.some((one) => !Number.isInteger(one) || one < 1 || one > SHEETS_IN_KIT)) {
    throw new Error(`"${text}" does not name sheets of a kit of ${SHEETS_IN_KIT}`)
  }
  return numbers
}

const changeWord = (word: string): string => (word === 'academic' ? 'acid' : 'academic')

/** The text to type for each named sheet. The sheet with a mistake has one word replaced by another word of the list. */
export const planSheetTexts = (
  sheets: readonly (readonly string[])[],
  numbers: readonly number[],
  mistakeSheet?: number,
): string[] => {
  if (mistakeSheet !== undefined && !numbers.includes(mistakeSheet)) {
    throw new Error(`the sheet ${mistakeSheet} is not among the sheets ${numbers.join(' and ')}`)
  }
  return numbers.map((number) => {
    const words = sheets.at(number - 1)
    if (words === undefined) throw new Error(`the kit has no sheet ${number}`)
    if (number !== mistakeSheet) return words.join(' ')
    return words.map((word, at) => (at === MISTAKE_AT ? changeWord(word) : word)).join(' ')
  })
}

/** Valid recovery words of an account other than the one of `own`. */
export const pickOtherWords = (own: readonly string[]): readonly string[] => {
  const other = OTHER_ACCOUNTS.find((words) => words.join(' ') !== own.join(' '))
  if (other === undefined) throw new Error('no other account is left to pick')
  return other
}

export const KIT_FILE = 'Numen recovery kit.pdf'

/**
 * The text each page of a PDF shows, in the order it is drawn: the strings of the text operators of
 * each content stream, inflated when it is compressed.
 */
export const readPdfPages = (pdf: Buffer): string[][] => {
  const pages: string[][] = []
  const text = pdf.toString('latin1')
  for (const found of text.matchAll(/stream\r?\n([\s\S]*?)\r?\nendstream/g)) {
    const raw = Buffer.from(found[1], 'latin1')
    let content: string
    try {
      content = zlib.inflateSync(raw).toString('latin1')
    } catch {
      content = raw.toString('latin1')
    }
    const shown = [...content.matchAll(/\(((?:[^()\\]|\\.)*)\) Tj/g)].map((one) => one[1])
    if (shown.length > 0) pages.push(shown)
  }
  return pages
}

/** The words of each sheet of a kit: the strings of a page that are one lower-case word, in order. */
export const readKitSheets = (pages: readonly (readonly string[])[]): string[][] =>
  pages.map((page) => page.filter((one) => /^[a-z]+$/.test(one)))

/** The place a field of the confirmation asks for: "Word 4" asks for the 4th word. */
export const readPosition = (label: string): number => {
  const found = /^Word (\d+)$/.exec(label)
  if (found?.[1] === undefined)
    throw new Error(`"${label}" is not the label of a confirmation of a word`)
  return Number(found[1])
}

/**
 * The words to type for the places the page asks about. A mistake replaces the first word that is not the mistake already,
 * so the answer is wrong whatever the words are.
 */
export const planAnswers = (
  positions: readonly number[],
  words: readonly string[],
  mistake?: string,
): string[] => {
  const answers = positions.map((position) => {
    const word = words.at(position - 1)
    if (word === undefined)
      throw new Error(`the page asks for the word ${position} of ${words.length}`)
    return word
  })
  if (mistake === undefined) return answers
  const at = answers.findIndex((word) => word !== mistake)
  if (at === -1) throw new Error(`every word asked for is "${mistake}": nothing can take its place`)
  return answers.map((word, index) => (index === at ? mistake : word))
}
