/** The rules of the driver that need no browser: they say what a scenario may name and fail on the rest. */
import path from 'node:path'

export type TabKind = 'PDF Reader' | 'EPUB Reader' | 'Audio Player'

export interface Reference {
  reference: 'at' | 'titled' | 'for the web address'
  name: string
}

const POSITIONS: Record<TabKind, string> = {
  'PDF Reader': 'page',
  'EPUB Reader': 'page',
  'Audio Player': 'time',
}

export const getStem = (file: string): string => path.basename(file).replace(/\.[^.]+$/, '')

export const getEscaped = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

export const getTitlePattern = (title: string): RegExp => new RegExp(`^(Linked tab )?${getEscaped(title)}( Close |$)`)

/** The title the product gives the tab of a web address: the characters a file name cannot have become "-". */
export const getAddressTitle = (address: string): string => address.replace(/[/\\:*?"<>|#]/g, '-')

export const parseReference = (reference: string, name: string): Reference => {
  if (reference !== 'at' && reference !== 'titled' && reference !== 'for the web address') {
    throw new Error(`"${reference}" does not name a tab or a note: use "at", "titled" or "for the web address"`)
  }
  return { reference, name }
}

export const assertPosition = (kind: TabKind, position: string, file: string): void => {
  if (POSITIONS[kind] !== position) {
    throw new Error(`the tab "${file}" is a ${kind} tab: its position is a ${POSITIONS[kind]}, not a ${position}`)
  }
}

export const parseTime = (time: string): number => {
  const match = /^(\d+):([0-5]\d)$/.exec(time)
  if (!match) throw new Error(`"${time}" is not a time: write minutes and seconds, like "2:00"`)
  return Number(match[1]) * 60 + Number(match[2])
}

export const assertEntry = (entry: string, path: string, isFolder: boolean): void => {
  if (entry !== 'file' && entry !== 'folder') throw new Error(`"${entry}" is not a file or a folder`)
  if ((entry === 'folder') !== isFolder) {
    throw new Error(`"${path}" is a ${isFolder ? 'folder' : 'file'}, not a ${entry}`)
  }
}

export const getAncestors = (entryPath: string): string[] => {
  const parts = entryPath.split('/').slice(0, -1)
  return parts.map((_, index) => parts.slice(0, index + 1).join('/'))
}

export type PaletteField = 'name' | 'item' | 'note' | 'text'

const PALETTE_FIELDS: Record<string, PaletteField> = { Name: 'name', Search: 'item', 'Search notes': 'note' }

/** What the field of the open palette asks for, read from its placeholder: a name, one item of a list, a note found by words, or a text. */
export const getPaletteField = (placeholder: string): PaletteField => PALETTE_FIELDS[placeholder] ?? 'text'

export const assertApplication = (name: string): void => {
  if (name !== 'editor' && name !== 'flashcards') {
    throw new Error(`"${name}" is not an application: use "editor" or "flashcards"`)
  }
}

const RATINGS = new Map([
  ['Again', 1],
  ['Hard', 2],
  ['Good', 3],
  ['Easy', 4],
])

/** The number the review log writes for an answer. */
export const getRating = (answer: string): number => {
  const rating = RATINGS.get(answer)
  if (rating === undefined) throw new Error(`"${answer}" is not an answer: use Again, Hard, Good or Easy`)
  return rating
}

export const parseReviewScope = (scope: string): 'vault' | 'deck' => {
  if (scope !== 'vault' && scope !== 'deck') throw new Error(`"${scope}" is not reviewed: use "vault" or "deck"`)
  return scope
}

export const parseCount = (text: string): number => {
  if (!/^\d+$/.test(text)) throw new Error(`"${text}" is not a number`)
  return Number(text)
}

export const readNoteType = (content: string): string => {
  const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/.exec(content)
  const match = /^type:\s*(\S+)\s*$/m.exec(frontmatter?.[1] ?? '')
  return (match?.[1] ?? '').replace(/^["']|["']$/g, '')
}

export const pickByTitle = (title: string, paths: readonly string[]): string => {
  const found = paths.filter((one) => getStem(one) === title)
  if (found.length === 0) throw new Error(`no tab is titled "${title}": use "at" with its path`)
  if (found.length > 1) throw new Error(`more than one tab is titled "${title}": use "at" with its path`)
  return found[0] as string
}

export const assertSingleTab = (title: string, count: number): void => {
  if (count > 1) throw new Error(`${count} tabs are titled "${title}": use "at" with the path`)
}
