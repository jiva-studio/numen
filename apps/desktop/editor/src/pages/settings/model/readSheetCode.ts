import { WORDS as words } from '../words'

const PREFIX = 'numen://recover?'
const WORDS_ON_SHEET = 33

export interface SheetCode {
  readonly kit: string
  readonly sheet: number
  readonly words: readonly string[]
}

export type SheetCodeReading =
  | { readonly code: SheetCode; readonly problem?: undefined }
  | { readonly code?: undefined; readonly problem: string }

/** What the QR code of a kit sheet says: its kit, its number and its 33 words. Anything else is no sheet code. */
export function readSheetCode(text: string): SheetCodeReading {
  const bad = { problem: words.scanNotSheet }
  const trimmed = text.trim()
  if (!trimmed.startsWith(PREFIX)) return bad
  const query = new URLSearchParams(trimmed.slice(PREFIX.length))
  if (query.get('v') !== '1') return bad
  const kit = query.get('kit') ?? ''
  const sheet = Number(query.get('sheet'))
  const list = (query.get('words') ?? '').split(/\s+/).filter((word) => word !== '')
  const isSheetNumber = Number.isInteger(sheet) && sheet >= 1 && sheet <= 3
  if (kit === '' || !isSheetNumber || list.length !== WORDS_ON_SHEET) return bad
  return { code: { kit, sheet, words: list.map((word) => word.toLowerCase()) } }
}

export interface ScannedSheets {
  readonly kit: string
  readonly sheets: readonly number[]
}

export type ScanStep =
  | { readonly next: ScannedSheets; readonly line: string; readonly problem?: undefined }
  | { readonly next?: undefined; readonly line?: undefined; readonly problem: string }

/** Adds a scanned sheet to the sheets scanned so far: a sheet of another kit or one scanned twice is refused. */
export function addScannedSheet(before: ScannedSheets | undefined, text: string): ScanStep {
  const read = readSheetCode(text)
  if (read.code === undefined) return { problem: read.problem }
  const { code } = read
  if (before !== undefined && before.kit !== code.kit) return { problem: words.scanOtherKit }
  if (before?.sheets.includes(code.sheet)) return { problem: words.scanRepeated(code.sheet) }
  return {
    next: { kit: code.kit, sheets: [...(before?.sheets ?? []), code.sheet] },
    line: code.words.join(' '),
  }
}
