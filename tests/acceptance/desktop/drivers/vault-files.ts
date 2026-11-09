/** The files of a table in a scenario: rows in, the writes to a vault out, and the checks of a vault out of its rows. */
import path from 'node:path'
import type { SampleKind } from '../support/samples.js'
import { assertVaultPath } from './rules.js'

export type VaultWrite =
  | { readonly kind: 'folder'; readonly path: string }
  | { readonly kind: 'note'; readonly path: string; readonly text: string }
  | { readonly kind: 'sample'; readonly path: string; readonly sample: string; readonly sampleKind: SampleKind }
  | { readonly kind: 'empty'; readonly path: string }

export interface VaultCheck {
  readonly path: string
  readonly isFolder: boolean
  readonly text: string
}

const FILE_COLUMNS = ['path', 'text', 'sample']
const CHECK_COLUMNS = ['path', 'text']

const SAMPLE_KINDS: Readonly<Record<string, SampleKind>> = {
  '.pdf': 'PDF',
  '.epub': 'EPUB',
  '.wav': 'recording',
  '.mp3': 'recording',
  '.flac': 'recording',
  '.png': 'image',
}

/** The kind of sample a file name stands for, by its extension. */
export const getSampleKind = (fileName: string): SampleKind => {
  const kind = SAMPLE_KINDS[path.extname(fileName).toLowerCase()] as SampleKind | undefined
  if (kind === undefined) {
    throw new Error(`"${fileName}" is not a sample: a sample is a ${Object.keys(SAMPLE_KINDS).join(', ')} file`)
  }
  return kind
}

/** The path as the table writes it: a folder ends with a slash. */
export const describePath = (write: Pick<VaultWrite, 'kind' | 'path'>): string =>
  write.kind === 'folder' ? `${write.path}/` : write.path

/** The columns of a table by their names: the allowed ones, in their order, each at most once, the first required. */
const readColumns = (header: readonly string[], allowed: readonly string[], what: string): string[] => {
  const list = allowed.join(' | ')
  for (const name of header) {
    if (!allowed.includes(name)) throw new Error(`"${name}" is not a column of ${what}: use ${list}`)
  }
  if (header[0] !== allowed[0]) throw new Error(`${what} starts with the column "${allowed[0]}": the columns are ${list}`)
  const places = header.map((name) => allowed.indexOf(name))
  if (places.some((place, index) => index > 0 && place <= places[index - 1])) {
    throw new Error(`the columns of ${what} are in the order ${list}, each once`)
  }
  return [...header]
}

const readTable = (rows: readonly string[][], allowed: readonly string[], what: string): Record<string, string>[] => {
  const [header = [], ...body] = rows
  const columns = readColumns(header, allowed, what)
  if (body.length === 0) throw new Error(`${what} has no row`)
  return body.map((row) => Object.fromEntries(columns.map((name, index) => [name, row[index] ?? ''])))
}

const parseWrite = (cells: Record<string, string>): VaultWrite => {
  const { path: written, text = '', sample = '' } = cells
  if (written === '') throw new Error('a row of the files has no path')
  if (text !== '' && sample !== '') throw new Error(`"${written}" has a text and a sample: give one of them`)
  if (written.endsWith('/')) {
    if (text !== '' || sample !== '') throw new Error(`"${written}" is a folder: a folder has no text and no sample`)
    assertVaultPath(written.slice(0, -1))
    return { kind: 'folder', path: written.slice(0, -1) }
  }
  assertVaultPath(written)
  if (sample !== '') return { kind: 'sample', path: written, sample, sampleKind: getSampleKind(sample) }
  if (text === '') return { kind: 'empty', path: written }
  if (!written.endsWith('.md')) throw new Error(`"${written}" has a text but is not a .md file: only a note has a text`)
  return { kind: 'note', path: written, text }
}

/** The writes a table of files asks for: folders, notes, samples and empty files. */
export const parseVaultFiles = (rows: readonly string[][]): VaultWrite[] => {
  const writes = readTable(rows, FILE_COLUMNS, 'the files table').map(parseWrite)
  const names = writes.map(describePath)
  const repeated = names.find((one, index) => names.indexOf(one) !== index)
  if (repeated !== undefined) throw new Error(`"${repeated}" is in the files twice`)
  return writes
}

/** Fails on the first path of the writes that the vault has. */
export const assertPathsAreFree = (writes: readonly VaultWrite[], isTaken: (entryPath: string) => boolean): void => {
  const taken = writes.find((one) => isTaken(one.path))
  if (taken !== undefined) throw new Error(`"${describePath(taken)}" is in the vault already`)
}

/** The checks a table asks of a vault: each path is there, a folder when it ends with a slash, a note with the text when it has one. */
export const parseVaultChecks = (rows: readonly string[][]): VaultCheck[] =>
  readTable(rows, CHECK_COLUMNS, 'the files table').map((cells) => {
    const { path: written, text = '' } = cells
    if (written === '') throw new Error('a row of the files has no path')
    const isFolder = written.endsWith('/')
    if (isFolder && text !== '') throw new Error(`"${written}" is a folder: a folder has no text`)
    const entryPath = isFolder ? written.slice(0, -1) : written
    assertVaultPath(entryPath)
    return { path: entryPath, isFolder, text }
  })
