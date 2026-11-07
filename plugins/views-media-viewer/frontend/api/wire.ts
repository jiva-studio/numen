/**
 * Wire adapter for media, the creation of text and plugin tasks; the transcript is
 * read and edited, and the properties of a note read, through the plugin SDK.
 */
import {
  resolveAddress,
  type CommandsHost,
  type TaskHost,
  type VaultNotes,
  type VaultTexts,
} from '@numen/plugin-sdk'
import type { Recordings } from '../types'
import { createSources } from './sources'
import { createTextCommand } from './text'
import { createWorkWatcher } from './work'
import { createTranscripts } from './transcripts'

let taskHost: TaskHost | undefined

/** Sets the task host the plugin was activated with. */
export const setTaskHost = (host: TaskHost | undefined) => {
  taskHost = host
}

/** The texts of the vault, as the plugin's context gave them when it was activated. */
let vaultTexts: VaultTexts | undefined

/** Keeps the texts the plugin's context gives, which the transcript is read and edited through. */
export const bindTexts = (texts: VaultTexts | undefined): void => {
  vaultTexts = texts
}

/** The notes of the vault, as the plugin's context gave them when it was activated. */
let vaultNotes: VaultNotes | undefined

/** Keeps the notes the plugin's context gives, which the media of a note are read through. */
export const bindNotes = (notes: VaultNotes | undefined): void => {
  vaultNotes = notes
}

/** The commands of the window, as the plugin's context gave them when it was activated. */
let commandsHost: CommandsHost | undefined

/** Keeps the commands the plugin's context gives, which the text of a recording is created through. */
export const bindCommands = (commands: CommandsHost | undefined): void => {
  commandsHost = commands
}

// Without a task host there is nothing to unsubscribe.
const work = createWorkWatcher(
  (onTasks) => taskHost?.watchAll(onTasks) ?? { dispose: () => undefined },
)

export const recordings: Recordings = {
  ...createSources(() => vaultNotes, resolveAddress),
  ...createTranscripts(() => vaultTexts),
  ...createTextCommand(() => commandsHost),
  watchWork: (path, onChange) => work.watch(path, onChange),
  watchChanges: (path, onChange) => work.watchChanges(path, onChange),
}

/** Tells the open recording tabs on these paths that the vault changed them. */
export const refreshRecordings = (paths: readonly string[]): void => {
  work.refresh(paths)
}
