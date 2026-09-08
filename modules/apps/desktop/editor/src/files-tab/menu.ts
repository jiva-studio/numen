/**
 * The menu on a row: what it offers, and what choosing an item comes to.
 *
 * A row standing for a note offers every command over a note that has a row of
 * its own in the palette, beside the three the tab does itself; any other row
 * offers what can be done to a file, and a recording or a scanned document the
 * run over it.
 */
import type { MenuItem } from '@numen/ui'
import { commandsOf, overNote } from '../shared/command/commands'
import type { Source } from '../shared/core'
import { WORDS as words } from '../shared/words'
import { WORDS as own } from './words'

/** What the tab does itself: the files it makes, and a name put in a field. */
export const NEW_NOTE = 'newNote'
export const NEW_DECK = 'newDeck'
export const NEW_STENCIL = 'newStencil'
export const NEW_PRESET = 'newPreset'
export const NEW_FOLDER = 'newFolder'
export const RENAME = 'rename'

/** The groups the items stand in, in the order they are drawn. */
const GROUP = {
  open: 'open',
  file: 'file',
  run: 'run',
  plex: 'plex',
  agent: 'agent',
  remove: 'remove',
} as const

/** The commands over a note, in the words the palette offers each of them by. */
const NOTED: ReadonlyMap<string, string> = new Map(
  overNote(commandsOf(words)).map((one) => [one.id, one.text]),
)

/** One command over a note, in the group it stands in. */
const noted = (id: string, group: string): readonly MenuItem[] => {
  const text = NOTED.get(id)
  return text === undefined ? [] : [{ id, text, group }]
}

/** What the tab makes, offered wherever the menu was asked for. */
const MADE: readonly MenuItem[] = [
  { id: NEW_NOTE, text: own.newNote, group: GROUP.file },
  { id: NEW_DECK, text: own.newDeck, group: GROUP.file },
  { id: NEW_STENCIL, text: own.newStencil, group: GROUP.file },
  { id: NEW_PRESET, text: own.newPreset, group: GROUP.file },
  { id: NEW_FOLDER, text: own.newFolder, group: GROUP.file },
]

/** What the tab does itself, offered on every row. */
const OWN: readonly MenuItem[] = [...MADE, { id: RENAME, text: own.rename, group: GROUP.file }]

/** What a row standing for a note offers. */
const NOTE: readonly MenuItem[] = [
  ...noted('read', GROUP.open),
  ...noted('travel', GROUP.open),
  ...OWN,
  ...noted('copy', GROUP.file),
  ...noted('child', GROUP.plex),
  ...noted('parent', GROUP.plex),
  ...noted('jump', GROUP.plex),
  ...noted('title', GROUP.plex),
  ...noted('ask', GROUP.agent),
  ...noted('remove', GROUP.remove),
]

/**
 * What a row standing for anything but a note offers, with the runs its kind
 * can be put through.
 */
const filed = (...runs: readonly MenuItem[]): readonly MenuItem[] => [
  ...OWN,
  ...noted('copy', GROUP.file),
  ...runs,
  { id: 'remove', text: own.remove, group: GROUP.remove },
]

const FILED = filed()

/** The run a recording can be put through, and the one a scan can. */
const TRANSCRIBE: MenuItem = { id: 'transcribe', text: own.transcribe, group: GROUP.run }
const RECOGNISE: MenuItem = { id: 'recognise', text: own.recognise, group: GROUP.run }

/**
 * The runs a url can be put through. It carries two things — the text at its
 * address and a copy of what is there — and each is downloaded and deleted on its
 * own.
 */
const DOWNLOAD_TEXT: MenuItem = { id: 'downloadText', text: own.downloadText, group: GROUP.run }
const DOWNLOAD_COPY: MenuItem = { id: 'downloadCopy', text: own.downloadCopy, group: GROUP.run }
const DELETE_TEXT: MenuItem = { id: 'deleteText', text: own.deleteText, group: GROUP.run }
const DELETE_COPY: MenuItem = { id: 'deleteCopy', text: own.deleteCopy, group: GROUP.run }

/** The run offered where this build can do it, and the file's own items alone where it cannot. */
const runnable = (run: MenuItem, canRun: RunGuard): readonly MenuItem[] =>
  canRun(run.id) ? filed(run) : FILED

/**
 * What a url offers: everything a file offers, and the runs over what is at the
 * address. A build that cannot reach one offers none of them, which is how a
 * person is told this build does not do it.
 */
const urls = (canRun: RunGuard): readonly MenuItem[] =>
  filed(
    ...(canRun(DOWNLOAD_TEXT.id) ? [DOWNLOAD_TEXT] : []),
    ...(canRun(DOWNLOAD_COPY.id) ? [DOWNLOAD_COPY] : []),
    ...(canRun(DELETE_TEXT.id) ? [DELETE_TEXT] : []),
    ...(canRun(DELETE_COPY.id) ? [DELETE_COPY] : []),
  )

/** What a selection of several offers, which is what means something for all of them. */
const SEVERAL: readonly MenuItem[] = [{ id: 'remove', text: own.remove, group: GROUP.remove }]

/** The row a menu was asked for on: what the vault holds there. */
export interface MenuRow {
  readonly source: Source
  readonly folder: boolean
}

/** Whether the window the menu is drawn in can do a run at all. */
export type RunGuard = (run: string) => boolean

/**
 * What the menu offers, in the order it is drawn: something to make where it
 * was asked off every row, removal over a selection of several, and everything
 * that can be done to the one row it was asked for on.
 */
export const itemsFor = (
  on: MenuRow | null,
  several: boolean,
  canRun: RunGuard,
): readonly MenuItem[] => {
  if (!on) return MADE
  if (several) return SEVERAL
  if (on.folder) return FILED
  if (on.source === 'note') return NOTE
  if (on.source === 'url') return urls(canRun)
  if (on.source === 'recording') return runnable(TRANSCRIBE, canRun)
  return on.source === 'book' ? runnable(RECOGNISE, canRun) : FILED
}

/** What the menu offers anywhere. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set(
  [...NOTE, ...filed(TRANSCRIBE, RECOGNISE), ...MADE, DOWNLOAD_TEXT, DOWNLOAD_COPY, DELETE_TEXT, DELETE_COPY].map(
    (one) => one.id,
  ),
)
