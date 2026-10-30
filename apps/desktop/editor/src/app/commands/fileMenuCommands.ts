/**
 * The commands of the menu of the file in front: the commands the enabled
 * plugins declare that apply to it now. A command over the file is the one the
 * palette lists from the manifests; a command over a version of the file is
 * registered here, one for each version.
 */
import { computed, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import { registerCommand, type Command } from '@/features/command-palette'
import { getFileMenu, runFileMenuEntry, type FileMenuEntry } from '@/entities/text-version'
import { WORDS } from '@/shared/words'
import { getNamedIcon } from '@/shared/icons'
import { getPluginCommandId } from './declaredCommands'

/** Whether the entry runs over a version of the file, and not over the file. */
export const isOverVersion = (entry: FileMenuEntry): boolean => entry.versionId !== ''

/** What the palette files an entry under: the declared command it is, or the version it is over. */
export const getFileMenuId = (entry: FileMenuEntry): string =>
  isOverVersion(entry)
    ? `file-menu:${entry.pluginId}:${entry.command}:${entry.versionId}`
    : getPluginCommandId(entry.pluginId, entry.command)

/**
 * The commands for the entries of the menu of the file at `file`, offered over that file alone. The
 * plugin name is the detail of an entry whose title another entry has.
 */
export function createFileMenuCommands(
  file: string,
  entries: readonly FileMenuEntry[],
): readonly Command[] {
  const counts = new Map<string, number>()
  for (const entry of entries) counts.set(entry.title, (counts.get(entry.title) ?? 0) + 1)
  return entries.map((entry): Command => ({
    id: getFileMenuId(entry),
    text: entry.title,
    ...((counts.get(entry.title) ?? 0) > 1 && entry.pluginName ? { detail: entry.pluginName } : {}),
    ...(entry.icon ? { icon: getNamedIcon(entry.icon) } : {}),
    group: 'file',
    isAvailable: (target) => target.isReady && target.file !== '' && target.file === file,
  }))
}

export interface FileMenuCommandsDeps {
  readonly get?: typeof getFileMenu
  readonly run?: typeof runFileMenuEntry
  readonly onRun?: (file: string) => void
  /** Receives the failure of a menu read that is not awaited. */
  readonly onError?: (error: unknown) => void
}

/**
 * The menu of one file, read by `load`. A menu read before is shown immediately
 * and read again, so plugin changes appear on the next load. Each entry over a
 * version has a handler that runs it and reports the result.
 */
export function useFileMenuCommands(deps: FileMenuCommandsDeps = {}) {
  const get = deps.get ?? getFileMenu
  const run = deps.run ?? runFileMenuEntry
  const menu = shallowRef<{ readonly file: string; readonly entries: readonly FileMenuEntry[] }>({
    file: '',
    entries: [],
  })
  const entriesByPath = new Map<string, readonly FileMenuEntry[]>()
  const pendingPaths = new Set<string>()
  let unregisterAll: readonly (() => void)[] = []
  let currentFile = ''

  const register = (entries: readonly FileMenuEntry[]) => {
    for (const unregister of unregisterAll) unregister()
    unregisterAll = entries.filter(isOverVersion).map((entry) =>
      registerCommand(getFileMenuId(entry), async (invocation, on) => {
        try {
          await run(invocation.file, entry)
          deps.onRun?.(invocation.file)
          on.writeMessage(WORDS.commandRan(entry.title), 'info')
        } catch (error) {
          on.writeMessage(formatErrorMessage(error), 'error')
        }
        entriesByPath.delete(invocation.file)
        await load(invocation.file).catch((error) => deps.onError?.(error))
      }),
    )
  }

  const show = (file: string, entries: readonly FileMenuEntry[]) => {
    register(entries)
    menu.value = { file, entries }
  }

  const isShown = (file: string, entries: readonly FileMenuEntry[]): boolean =>
    menu.value.file === file && JSON.stringify(menu.value.entries) === JSON.stringify(entries)

  const load = async (file: string): Promise<void> => {
    currentFile = file
    if (!file) {
      show('', [])
      return
    }
    const cached = entriesByPath.get(file)
    if (cached && !isShown(file, cached)) show(file, cached)
    if (pendingPaths.has(file)) return
    pendingPaths.add(file)
    try {
      const entries = await get(file)
      entriesByPath.set(file, entries)
      if (currentFile === file && !isShown(file, entries)) show(file, entries)
    } finally {
      pendingPaths.delete(file)
    }
  }

  const getForFile = (file: string): readonly Command[] => {
    if (file) runInBackground(load(file), (error) => deps.onError?.(error))
    const entries = entriesByPath.get(file)
    return entries ? createFileMenuCommands(file, entries) : []
  }

  const getIcon = (id: string) =>
    createFileMenuCommands(menu.value.file, menu.value.entries).find((one) => one.id === id)?.icon

  return {
    paletteCommands: computed(() =>
      createFileMenuCommands(menu.value.file, menu.value.entries.filter(isOverVersion)),
    ),
    load,
    getForFile,
    getIcon,
  }
}

export type FileMenuCommands = ReturnType<typeof useFileMenuCommands>
