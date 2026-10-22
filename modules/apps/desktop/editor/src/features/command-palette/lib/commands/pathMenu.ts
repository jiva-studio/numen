/**
 * The window's commands over one row of the vault, as the menu on a row of the
 * files tree offers them.
 *
 * A row standing for a note offers every command over a note that has a row of
 * its own in the palette; any other row offers what can be done to a file, and
 * a recording or a scanned document the run over it.
 */
import type { MenuItem } from '@numen/ui'
import type { ArtifactState, ArtifactStates } from '@/entities/artifact'
import type { BookFormat, DocumentFormat, Source } from '@/entities/file'
import { WORDS as words } from '@/shared/words'
import { getNoteCommands } from './commands'
import { createCommands } from './table'
import { PATH_WORDS as own } from '../words'

/** One command the menu offers, in the group it is drawn in. */
export type PathItem = MenuItem & { readonly group: string }

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
  getNoteCommands(createCommands(words)).map((one) => [one.id, one.text]),
)

const getNoteCommand = (id: string, group: string): readonly PathItem[] => {
  return [{ id, text: NOTED.get(id) ?? id, group }]
}

/** What a row standing for a note offers. */
const NOTE: readonly PathItem[] = [
  ...getNoteCommand('read', GROUP.open),
  ...getNoteCommand('travel', GROUP.open),
  ...getNoteCommand('copy', GROUP.file),
  ...getNoteCommand('child', GROUP.plex),
  ...getNoteCommand('parent', GROUP.plex),
  ...getNoteCommand('jump', GROUP.plex),
  ...getNoteCommand('title', GROUP.plex),
  ...getNoteCommand('ask', GROUP.agent),
  ...getNoteCommand('remove', GROUP.remove),
]

/** The removal of a file or of several. */
const REMOVE: PathItem = { id: 'remove', text: own.remove, group: GROUP.remove }

/**
 * What a row standing for anything but a note offers, with the runs its kind
 * can be put through.
 */
const getFileMenu = (...runs: readonly PathItem[]): readonly PathItem[] => [
  ...getNoteCommand('copy', GROUP.file),
  ...runs,
  REMOVE,
]

const FILED = getFileMenu()

/** The runs a recording or a scan can be put through. */
const TRANSCRIBE: PathItem = { id: 'transcribe', text: own.transcribe, group: GROUP.run }
const RECOGNISE: PathItem = { id: 'recognise', text: own.recognise, group: GROUP.run }
const PROOFREAD: PathItem = { id: 'proofread', text: own.proofread, group: GROUP.run }

/**
 * The runs a url can be put through. It carries two things — the text at its
 * address and a copy of what is there — and each is downloaded and deleted on its
 * own.
 */
const DOWNLOAD_TEXT: PathItem = { id: 'downloadText', text: own.downloadText, group: GROUP.run }
const DOWNLOAD_COPY: PathItem = { id: 'downloadCopy', text: own.downloadCopy, group: GROUP.run }
const DELETE_TEXT: PathItem = { id: 'deleteText', text: own.deleteText, group: GROUP.run }
const DELETE_COPY: PathItem = { id: 'deleteCopy', text: own.deleteCopy, group: GROUP.run }

const isBusyOrDone = (state: ArtifactState | undefined): boolean =>
  state === 'done' || state === 'running' || state === 'queued'

const isProofreadDisabled = (
  sourceState: ArtifactState | undefined,
  proofreadState: ArtifactState | undefined,
): boolean => sourceState !== 'done' || proofreadState === 'done' || proofreadState === 'running'

/** What a recording offers: transcribe and proofread, disabled once done. */
const getRecordingRuns = (
  createdItem: ArtifactStates | undefined,
  canRun: RunGuard,
): readonly PathItem[] => {
  const runs: PathItem[] = []
  if (canRun(TRANSCRIBE.id)) {
    runs.push({ ...TRANSCRIBE, isDisabled: isBusyOrDone(createdItem?.transcript) })
  }
  if (canRun(PROOFREAD.id)) {
    runs.push({
      ...PROOFREAD,
      isDisabled: isProofreadDisabled(
        createdItem?.transcript,
        createdItem?.['transcript.corrected'],
      ),
    })
  }
  return runs
}

/** What a book offers: recognise and proofread, disabled once done. */
const getBookRuns = (
  format: DocumentFormat | BookFormat | undefined,
  createdItem: ArtifactStates | undefined,
  canRun: RunGuard,
): readonly PathItem[] => {
  if (format === 'epub') return []
  const runs: PathItem[] = []
  if (canRun(RECOGNISE.id)) {
    runs.push({ ...RECOGNISE, isDisabled: isBusyOrDone(createdItem?.ocr) })
  }
  if (canRun(PROOFREAD.id)) {
    runs.push({
      ...PROOFREAD,
      isDisabled: isProofreadDisabled(createdItem?.ocr, createdItem?.['ocr.corrected']),
    })
  }
  return runs
}

/**
 * What a url offers: everything a file offers, and the runs over what is at the
 * address. A build that cannot reach one offers none of them, which is how a
 */
const urls = (canRun: RunGuard): readonly PathItem[] =>
  getFileMenu(
    ...(canRun(DOWNLOAD_TEXT.id) ? [DOWNLOAD_TEXT] : []),
    ...(canRun(DOWNLOAD_COPY.id) ? [DOWNLOAD_COPY] : []),
    ...(canRun(DELETE_TEXT.id) ? [DELETE_TEXT] : []),
    ...(canRun(DELETE_COPY.id) ? [DELETE_COPY] : []),
  )

/** The row a menu was asked for on: what the vault holds there. */
export interface MenuRow {
  readonly source: Source
  readonly isFolder: boolean
  readonly format?: DocumentFormat | BookFormat | undefined
  readonly made?: ArtifactStates | undefined
}

/** Whether the window the menu is drawn in can do a run at all. */
export type RunGuard = (run: string) => boolean

/**
 * The window's commands over a row, in the order they are drawn: removal over a
 * selection of several, and everything that can be done to the one row.
 */
export const createPathCommands = (
  on: MenuRow,
  hasSeveral: boolean,
  canRun: RunGuard,
): readonly PathItem[] => {
  if (hasSeveral) return [REMOVE]
  if (on.isFolder) return FILED
  if (on.source === 'note') return NOTE
  if (on.source === 'url') return urls(canRun)
  if (on.source === 'recording') return getFileMenu(...getRecordingRuns(on.made, canRun))
  if (on.source === 'book') return getFileMenu(...getBookRuns(on.format, on.made, canRun))
  return FILED
}
