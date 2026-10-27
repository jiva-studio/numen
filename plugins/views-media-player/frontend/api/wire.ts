/**
 * Wire adapter for media, file menu and plugin tasks; the transcript is
 * read and edited through the plugin SDK.
 */
import {
  getFileMenu,
  getMediaSource,
  readFileMenuEntries,
  runFileMenuEntry,
  writeFileMenuAction,
  type TaskHost,
  type VaultTexts,
} from '@numen/plugin-sdk'
import type { Recordings } from '../types'
import { createWorkWatcher } from './work'
import { createTranscripts } from './transcripts'

let taskHost: TaskHost | undefined

/** Sets the task host the plugin was activated with. */
export const setTaskHost = (host: TaskHost | undefined) => {
  taskHost = host
}

/** The action of the schema an entry's action is. */
export const writeMenuAction = writeFileMenuAction

/** The entries of the schema the tab reads, leaving out an action it does not know. */
export const readMenuEntries = readFileMenuEntries

/** The texts of the vault, as the plugin's context gave them when it was activated. */
let vaultTexts: VaultTexts | undefined

/** Keeps the texts the plugin's context gives, which the transcript is read and edited through. */
export const bindTexts = (texts: VaultTexts | undefined): void => {
  vaultTexts = texts
}

const work = createWorkWatcher((onTasks) => taskHost?.watchAll(onTasks) ?? { dispose: () => {} })

export const recordings: Recordings = {
  getSource: (path) => getMediaSource(path),
  ...createTranscripts(() => vaultTexts),
  getMenu: (path) => getFileMenu(path),
  runMenuEntry: (path, entry) => runFileMenuEntry(path, entry),
  watchWork: (path, onChange) => work.watch(path, onChange),
}
