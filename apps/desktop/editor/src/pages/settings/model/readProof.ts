import type { KitProof } from '@/entities/identity'
import { WORDS as words } from '../words'

const WORDS_IN_PHRASE = 12
const WORDS_ON_SHEET = 33
const SHEETS_NEEDED = 2

export interface ProofReading {
  readonly proof: KitProof | undefined
  readonly note: string
  readonly isProblem: boolean
}

const splitWords = (text: string): string[] =>
  text
    .split(/\s+/)
    .filter((word) => word !== '')
    .map((word) => word.toLowerCase())

const createReading = (note: string, isProblem = false, proof?: KitProof): ProofReading => ({
  proof,
  note,
  isProblem,
})

function readPhrase(entered: readonly string[]): ProofReading {
  const count = entered.length
  if (count < WORDS_IN_PHRASE) return createReading(words.proofWordsSoFar(count))
  return count === WORDS_IN_PHRASE
    ? createReading(words.proofWordsRecognized, false, { words: entered })
    : createReading(words.proofWrongCount(count), true)
}

function readSheetLine(sheets: readonly (readonly string[])[], at: number): ProofReading | null {
  const sheet = sheets[at] ?? []
  if (sheet.length > WORDS_ON_SHEET) {
    return createReading(words.proofSheetLong(at + 1, sheet.length), true)
  }
  if (sheet.length < WORDS_ON_SHEET) return createReading(words.proofSheetShort(at + 1, sheet.length))
  const first = sheets.findIndex((other) => other.join(' ') === sheet.join(' '))
  return first < at ? createReading(words.proofSheetRepeated(at + 1, first + 1), true) : null
}

function readSheets(sheets: readonly (readonly string[])[]): ProofReading {
  for (let at = 0; at < sheets.length; at += 1) {
    const found = readSheetLine(sheets, at)
    if (found) return found
  }
  if (sheets.length > SHEETS_NEEDED) return createReading(words.proofTooManySheets, true)
  if (sheets.length < SHEETS_NEEDED) return createReading(words.proofOneSheet(sheets.length))
  return createReading(words.proofTwoSheets, false, { sheets })
}

/**
 * What a text proves: the 12 recovery words, or one sheet of the recovery kit on each line. A text
 * of one line is the words unless it has the 33 words of a sheet; a text of several lines is sheets
 * unless every line is short enough to be a part of the words.
 */
export function readProof(text: string): ProofReading {
  const lines = text
    .split('\n')
    .map(splitWords)
    .filter((line) => line.length > 0)
  if (lines.length === 0) return createReading('')
  const isPhrase = lines.every((line) => line.length <= WORDS_IN_PHRASE)
  const only = lines.length === 1 ? (lines[0] ?? []) : []
  if (isPhrase) return readPhrase(lines.flat())
  if (only.length > 0 && only.length !== WORDS_ON_SHEET) {
    return createReading(words.proofWrongCount(only.length), true)
  }
  return readSheets(lines)
}
