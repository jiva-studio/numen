/** The rules of the driver that need no browser: they say what a scenario may name and fail on the rest. */
import path from 'node:path'

export type TabKind = 'PDF Reader' | 'EPUB Reader' | 'Audio Player'

export interface Reference {
  how: 'at' | 'titled'
  value: string
}

const PLACES: Record<TabKind, string> = {
  'PDF Reader': 'page',
  'EPUB Reader': 'page',
  'Audio Player': 'time',
}

export const getStem = (file: string): string => path.basename(file).replace(/\.[^.]+$/, '')

export const getEscaped = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** A title names a tab by its end: the tab of a link is titled by its whole address, and a person names its last words. */
export const getTitlePattern = (title: string): RegExp => new RegExp(`${getEscaped(title)}( Close |$)`)

export const parseReference = (how: string, value: string): Reference => {
  if (how !== 'at' && how !== 'titled') {
    throw new Error(`"${how}" does not name a tab or a note: use "at" or "titled"`)
  }
  return { how, value }
}

export const assertPlace = (kind: TabKind, place: string, file: string): void => {
  if (PLACES[kind] !== place) {
    throw new Error(`the tab "${file}" is a ${kind} tab: its place is a ${PLACES[kind]}, not a ${place}`)
  }
}

export const parseTime = (time: string): number => {
  const hit = /^(\d+):([0-5]\d)$/.exec(time)
  if (!hit) throw new Error(`"${time}" is not a time: write minutes and seconds, like "2:00"`)
  return Number(hit[1]) * 60 + Number(hit[2])
}

export type PaletteAsk = 'name' | 'item' | 'text'

const PALETTE_ASKS: Record<string, PaletteAsk> = { Name: 'name', Search: 'item' }

/** What the field of the open palette asks for, read from its placeholder: a name, one item of a list, or a text. */
export const parseAsk = (placeholder: string): PaletteAsk => PALETTE_ASKS[placeholder] ?? 'text'

export const assertApplication = (name: string): void => {
  if (name === 'flashcards') throw new Error('the application "flashcards" is not built yet: use "editor"')
  if (name !== 'editor') throw new Error(`"${name}" is not an application: use "editor"`)
}

export const parseCount = (text: string): number => {
  if (!/^\d+$/.test(text)) throw new Error(`"${text}" is not a number`)
  return Number(text)
}

export const readNoteType = (content: string): string => {
  const frontmatter = /^---\r?\n([\s\S]*?)\r?\n---/.exec(content)
  const hit = /^type:\s*(\S+)\s*$/m.exec(frontmatter?.[1] ?? '')
  return (hit?.[1] ?? '').replace(/^["']|["']$/g, '')
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
