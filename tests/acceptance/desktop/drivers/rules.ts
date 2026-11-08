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

const PALETTE_STEPS = ['name']

export const getStem = (file: string): string => path.basename(file).replace(/\.[^.]+$/, '')

export const getEscaped = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

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

export const assertPaletteStep = (kind: string): void => {
  if (!PALETTE_STEPS.includes(kind)) throw new Error(`the palette step "${kind}" is not built yet`)
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
