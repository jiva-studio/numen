/**
 * The commands offered over the note in front, and what carries each out.
 */
import { runInBackground } from '@numen/ui'
import { type Command, type FeatureCommands } from '@/entities/command'
import {
  ARGUMENT,
  COMMAND,
  ROW,
  createRowKeys,
  flushTab,
  withWhen,
  type CommandInvocation,
  type CommandPlace,
  type Notes,
} from '@/shared/model/command'
import { removeFile } from '@/entities/file'
import { renameNote, type NoteCreator } from '@/entities/note'
import { NOTE } from '@/entities/tab'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { ConnectionRole } from '@numen/plugin-loader'
import { formatNames, joinAll } from '../lib/voice'
import { NOTE_WORDS, type NoteWords } from '../words'

/** Where a command takes the window. */
interface NoteNavigator {
  /** Shows a note in the plugin that displays notes. */
  travel(path: string): Promise<void>
  /** Every tab standing on a note moves to another one. */
  leave(from: string, to: string): Promise<void>
  /** The note the vault opens with. */
  getOpeningNote(): string
  /** The tab of a kind last in front, or a new one, shown at a path. */
  showTab(kind: string, path: string): void
}

/** What the commands over a note use of the window. */
export interface NoteCommandsDeps {
  readonly making: NoteCreator
  readonly notes: Notes
  readonly navigator: NoteNavigator
  readonly writeMessage: MessageWriter
}

const getPlace = (invocation: CommandInvocation): CommandPlace | undefined => {
  const { selector, line } = invocation.target
  if (selector !== undefined) return { selector }
  return line === undefined ? undefined : { line }
}

const openFile = (invocation: CommandInvocation, deps: NoteCommandsDeps): void => {
  const file = invocation.target.path || invocation.target.file
  const placement = invocation.values?.[ARGUMENT.placement] === 'split' ? 'split' : 'current'
  deps.notes.openFile(file, invocation.target.title || file, placement, getPlace(invocation))
}

const revealFile = (invocation: CommandInvocation, deps: NoteCommandsDeps): void => {
  const kind = invocation.values?.[ARGUMENT.kind]
  if (kind) deps.navigator.showTab(kind, invocation.target.path)
}

const createNoteCommand = async (
  invocation: CommandInvocation,
  deps: NoteCommandsDeps,
  relation: ConnectionRole | null,
): Promise<string | void> => {
  if (!invocation.name) {
    if (invocation.target.isFolder) return deps.making.createUntitled(invocation.target.path)
    return
  }
  const made = await deps.making.createWithTitle(
    invocation.name,
    relation ? invocation.target.path : '',
    relation,
  )
  if (!made) return
  if (invocation.target.kind === NOTE) {
    deps.notes.openFile(made.path, made.title, 'split')
  } else if (made.path) {
    await deps.navigator.travel(made.path)
  } else {
    deps.writeMessage(NOTE_WORDS.nowhere, 'warning')
  }
  return made.path
}

const renameNoteCommand = async (
  invocation: CommandInvocation,
  deps: NoteCommandsDeps,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.target.title) return
  const tab = await flushTab(invocation.target.path, deps.notes)
  if (tab.isWaiting) return deps.writeMessage(NOTE_WORDS.unanswered, 'warning')
  const answer = await renameNote(invocation.target.path, invocation.name)
  if (answer.ok) return
  if (answer.error === 'changed') return deps.writeMessage(NOTE_WORDS.stale, 'warning')
  deps.writeMessage(NOTE_WORDS.errors[answer.error], 'error')
}

const removeFiles = async (
  invocation: CommandInvocation,
  deps: NoteCommandsDeps,
): Promise<void> => {
  const dangling: string[] = []
  const errors: string[] = []
  let isWaiting = false
  const opening = deps.navigator.getOpeningNote()

  for (const path of [invocation.target.path, ...invocation.target.others]) {
    const tab = await flushTab(path, deps.notes)
    if (tab.isWaiting) {
      isWaiting = true
      continue
    }
    const answer = await removeFile(path)
    if (!answer.ok) {
      errors.push(NOTE_WORDS.errors[answer.error])
      continue
    }
    if (tab.tabId) deps.notes.close(tab.tabId)
    dangling.push(...answer.value.dangling)
    if (opening) await deps.navigator.leave(path, opening)
  }

  if (errors.length > 0) return deps.writeMessage(joinAll(...errors), 'error')
  if (isWaiting) return deps.writeMessage(NOTE_WORDS.unanswered, 'warning')
  deps.writeMessage(formatNames(NOTE_WORDS.dangling, dangling), 'warning')
}

const copyPath = (invocation: CommandInvocation, deps: NoteCommandsDeps): void => {
  const clipboard = navigator.clipboard as Clipboard | undefined
  if (!clipboard) return
  runInBackground(clipboard.writeText(invocation.target.path), (error) =>
    deps.writeMessage(formatError(error), 'error'),
  )
}

const createOpenCommands = (words: NoteWords): readonly Command[] => [
  {
    id: COMMAND.openFile,
    text: words.read,
    group: 'note',
    ...withWhen('target.isNote'),
    also: ROW.openNoteBeside,
  },
  {
    id: ROW.openNoteBeside,
    command: COMMAND.openFile,
    values: { [ARGUMENT.placement]: 'split' },
    text: words.beside,
    group: 'note',
    ...withWhen('target.isNote'),
  },
  // The slots draw this command with the kind of place each entry names; the palette offers its rows.
  {
    id: COMMAND.revealFile,
    text: words.reveal,
    group: 'note',
    when: 'target.isNote',
    isAvailable: () => false,
  },
]

const createRows = (agent: string): readonly Command[] => [
  ...createOpenCommands(NOTE_WORDS),
  {
    id: COMMAND.createChildNote,
    text: NOTE_WORDS.child,
    ...createRowKeys(COMMAND.createChildNote, agent),
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.createParentNote,
    text: NOTE_WORDS.parent,
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.createJumpNote,
    text: NOTE_WORDS.jump,
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.renameNote,
    text: NOTE_WORDS.title,
    group: 'note',
    input: 'name',
    ...withWhen('target.isNote'),
    getFieldText: (target) => target.title,
  },
  // The note moves to the vault's .trash folder, and restoring it is a move.
  {
    id: COMMAND.removeFile,
    text: NOTE_WORDS.remove,
    group: 'note',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.copyFilePath,
    text: NOTE_WORDS.copy,
    group: 'note',
    ...withWhen('target.isNote'),
  },
  {
    id: COMMAND.createNote,
    text: NOTE_WORDS.newNote,
    ...createRowKeys(COMMAND.createNote, agent),
    group: 'window',
    iconName: 'file-plus',
    input: 'name',
    canOmitNameOverFolder: true,
    ...withWhen('window.isVaultReady'),
  },
]

/**
 * The commands over the note in front, and the one that makes a note. A command reached by Shift
 * and Enter on another one's row is declared too; the row it belongs to is the one that names it.
 */
export const createNoteCommands = (
  deps: NoteCommandsDeps,
  agent: string = navigator.userAgent,
): FeatureCommands => ({
  commands: createRows(agent),
  handlers: {
    [COMMAND.openFile]: (invocation) => openFile(invocation, deps),
    [COMMAND.revealFile]: (invocation) => revealFile(invocation, deps),
    [COMMAND.createChildNote]: (invocation) => createNoteCommand(invocation, deps, 'child'),
    [COMMAND.createParentNote]: (invocation) => createNoteCommand(invocation, deps, 'parent'),
    [COMMAND.createJumpNote]: (invocation) => createNoteCommand(invocation, deps, 'jump'),
    [COMMAND.createNote]: (invocation) => createNoteCommand(invocation, deps, null),
    [COMMAND.renameNote]: (invocation) => renameNoteCommand(invocation, deps),
    [COMMAND.removeFile]: (invocation) => removeFiles(invocation, deps),
    [COMMAND.copyFilePath]: (invocation) => copyPath(invocation, deps),
  },
})
