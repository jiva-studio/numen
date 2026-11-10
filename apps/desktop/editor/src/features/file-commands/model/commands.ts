/**
 * The commands over files and folders: make a folder, move a file, and open a document in a tab.
 */
import {
  COMMAND,
  ROW,
  flushTab,
  withWhen,
  type CommandInvocation,
  type FeatureCommands,
  type Notes,
} from '@/entities/command'
import { createFolder, listFiles, moveFile } from '@/entities/file'
import type { MessageWriter } from '@/shared/notices/messages'
import { generateUniqueName } from '@/shared/paths'
import { FILE_WORDS } from '../words'

/** What the commands over files use of the window. */
export interface FileCommandsDeps {
  readonly notes: Notes
  readonly writeMessage: MessageWriter
}

/**
 * A file or a folder filed somewhere else, carrying the name the path ends in. A destination that
 * is taken leaves it where it was.
 */
const moveFileCommand = async (
  invocation: CommandInvocation,
  deps: FileCommandsDeps,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.path) return
  const tab = await flushTab(invocation.path, deps.notes)
  if (tab.isWaiting) return deps.writeMessage(FILE_WORDS.unanswered, 'warning')
  const answer = await moveFile(invocation.path, invocation.name)
  if (answer.ok) return
  if (answer.error === 'alreadyExists') {
    return deps.writeMessage(FILE_WORDS.alreadyExists, 'error')
  }
  deps.writeMessage(FILE_WORDS.errors[answer.error], 'error')
}

const getUntitledFolderPath = async (invocation: CommandInvocation): Promise<string> => {
  if (!invocation.isFolder) return ''
  const names = (await listFiles(invocation.path)).map((one) => one.name)
  const name = generateUniqueName(names, FILE_WORDS.untitledFolder)
  return invocation.path ? `${invocation.path}/${name}` : name
}

/**
 * An empty folder, made under the path that was typed. Over a folder and without a path it is made
 * in that folder under a name nothing there carries. Answers the path of the folder made.
 */
const createFolderCommand = async (
  invocation: CommandInvocation,
  deps: FileCommandsDeps,
): Promise<string | void> => {
  const path = invocation.name || (await getUntitledFolderPath(invocation))
  if (!path) return
  const error = await createFolder(path)
  if (error === 'alreadyExists') return deps.writeMessage(FILE_WORDS.alreadyExists, 'error')
  if (error) return deps.writeMessage(FILE_WORDS.errors[error], 'error')
  return path
}

/**
 * The commands over files: make a folder, move a file, and open a document in a tab. Opening a
 * document runs the command that opens a file, which the commands over a note declare. Moving a
 * file has no row in the palette; a plugin runs it.
 */
export const createFileCommands = (deps: FileCommandsDeps): FeatureCommands => ({
  commands: [
    {
      id: COMMAND.createFolder,
      text: FILE_WORDS.newFolder,
      group: 'window',
      input: 'name',
      canOmitNameOverFolder: true,
      ...withWhen('window.isVaultReady'),
    },
    {
      id: ROW.openDocument,
      command: COMMAND.openFile,
      text: FILE_WORDS.readDocument,
      group: 'file',
      ...withWhen('target.isFile && !window.isTabOpen'),
    },
  ],
  handlers: {
    [COMMAND.createFolder]: (invocation) => createFolderCommand(invocation, deps),
    [COMMAND.moveFile]: (invocation) => moveFileCommand(invocation, deps),
  },
})
