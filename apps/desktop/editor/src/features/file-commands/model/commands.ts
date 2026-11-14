/**
 * The commands over files and folders: make a folder, move a file, and open a document in a tab.
 */
import { type FeatureCommands } from '@/entities/command'
import {
  COMMAND,
  ROW,
  flushTab,
  withWhen,
  type CommandInvocation,
  type Notes,
} from '@/shared/model/command'
import type { FileApi } from '@/entities/file'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { generateUniqueName } from '@/shared/lib/paths'
import { FILE_WORDS } from '../words'

/** What the commands over files use of the window. */
export interface FileCommandsDeps {
  readonly files: Pick<FileApi, 'createFolder' | 'listFiles' | 'moveFile'>
  readonly notes: Notes
  readonly writeMessage: MessageWriter
}

const moveFileCommand = async (
  invocation: CommandInvocation,
  deps: FileCommandsDeps,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.target.path) return
  const tab = await flushTab(invocation.target.path, deps.notes)
  if (tab.isWaiting) return deps.writeMessage(FILE_WORDS.unanswered, 'warning')
  const answer = await deps.files.moveFile(invocation.target.path, invocation.name)
  if (answer.ok) return
  if (answer.error === 'alreadyExists') {
    return deps.writeMessage(FILE_WORDS.alreadyExists, 'error')
  }
  deps.writeMessage(FILE_WORDS.errors[answer.error], 'error')
}

const getUntitledFolderPath = async (
  invocation: CommandInvocation,
  deps: FileCommandsDeps,
): Promise<string> => {
  if (!invocation.target.isFolder) return ''
  const existing = (await deps.files.listFiles(invocation.target.path)).map((one) => one.name)
  const name = generateUniqueName(existing, FILE_WORDS.untitledFolder)
  return invocation.target.path ? `${invocation.target.path}/${name}` : name
}

const createFolderCommand = async (
  invocation: CommandInvocation,
  deps: FileCommandsDeps,
): Promise<string | void> => {
  const path = invocation.name || (await getUntitledFolderPath(invocation, deps))
  if (!path) return
  const answer = await deps.files.createFolder(path)
  if (!answer.ok) {
    if (answer.error === 'alreadyExists')
      return deps.writeMessage(FILE_WORDS.alreadyExists, 'error')
    return deps.writeMessage(FILE_WORDS.errors[answer.error], 'error')
  }
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
