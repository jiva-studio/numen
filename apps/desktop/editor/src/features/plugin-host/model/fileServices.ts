/**
 * The files of the vault as a plugin's tabs are given them: the folders listed,
 * the files made and moved, and the icons the window draws commands and entries with.
 */
import type { Component } from 'vue'
import type { VaultFiles, WorkspaceIcons } from '@numen/plugin-loader'
import type { DownloadBytes, Downloads, Entry, FileApi } from '@/entities/file'
import { COMMAND, findIcon, type CommandTarget } from '@/shared/model/command'
import type { NoteCreator } from '@/entities/note'
import { getEntryIcon } from './window/entryIcon'

/** What the files of the vault are listed, made and moved through. */
export interface VaultFilesDeps {
  readonly listFiles: FileApi['listFiles']
  readonly downloads: Downloads
  readonly making: NoteCreator
  readonly getTarget: () => CommandTarget
  readonly runCommand: (id: string, target: CommandTarget, name: string) => Promise<void>
}

/** A file here carries the provider of the vault, which is what lets a person take its download off again. */
const markEntry = (
  one: Entry,
  provider: string,
  isDownloading: boolean,
  progress: DownloadBytes | undefined,
): Entry => ({
  ...one,
  ...(!one.isFolder && !one.isAbsent && provider ? { provider } : {}),
  ...(isDownloading ? { isDownloading: true } : {}),
  ...(isDownloading && progress
    ? { downloadProgress: progress.bytesDone / progress.bytesTotal }
    : {}),
})

export function createVaultFiles(deps: VaultFilesDeps): VaultFiles {
  const { making, getTarget, runCommand: run } = deps
  /** The plugin that fills the absent files of the vault, once a listing has shown one. */
  let provider = ''
  return {
    list: async (folder) => {
      const listed = await deps.listFiles(folder)
      provider = listed.find((one) => one.isAbsent && one.provider)?.provider ?? provider
      return listed.map((one) =>
        markEntry(
          one,
          provider,
          deps.downloads.isRunning(one.path),
          deps.downloads.getProgress(one.path),
        ),
      )
    },
    watchDownloads: (handler) => {
      const stop = deps.downloads.watch((path) => handler([path]))
      return { dispose: stop }
    },
    move: (from, to) => run(COMMAND.moveFile, { ...getTarget(), path: from }, to),
    createFolder: (path) => run(COMMAND.createFolder, getTarget(), path),
    createNote: (folder) => making.createUntitledNote(folder),
  }
}

/** What the window finds the icons of commands and files with. */
export interface IconHostDeps {
  readonly getPluginCommandIcon?: (id: string) => Component | undefined
  readonly findFileIcon?: ((path: string) => Component | undefined) | undefined
}

/** The icons the window draws commands and the entries of the vault with. */
export function createIconHost(deps: IconHostDeps): WorkspaceIcons {
  return {
    getCommand: (id) => deps.getPluginCommandIcon?.(id) ?? findIcon(id) ?? null,
    getEntry: (entry, isOpen) => getEntryIcon(entry, isOpen, deps.findFileIcon),
  }
}
