/**
 * The menu the text versions of the file in front make, offered in the
 * palette: the entry each plugin declares, under its title and icon, and
 * deleting a version, each by the plugin the application names.
 */
import { computed, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { registerCommand, type Command } from '@/features/command-palette'
import { getFileMenu, runFileMenuEntry, type FileMenuEntry } from '@/entities/text-version'
import { WORDS } from '@/shared/words'
import { getNamedIcon } from '@/shared/icons'

/** What the palette files an entry under: the action, the plugin and the version it is over. */
export const getFileMenuId = (entry: FileMenuEntry): string =>
  `file-menu:${entry.action}:${entry.pluginId}:${entry.versionId}`

/** What an entry is called in the palette, by the plugin that does it. */
export const getFileMenuText = (entry: FileMenuEntry): string =>
  entry.action === 'delete-version'
    ? WORDS.deleteVersionOf(entry.pluginName)
    : WORDS.runWith(entry.title, entry.pluginName)

/** The palette's commands for the entries of the menu of the file at `file`, offered over that file alone. */
export function createFileMenuPaletteCommands(
  file: string,
  entries: readonly FileMenuEntry[],
): readonly Command[] {
  return entries.map((entry): Command => ({
    id: getFileMenuId(entry),
    text: getFileMenuText(entry),
    ...(entry.action === 'create-text' ? { icon: getNamedIcon(entry.icon) } : {}),
    group: 'file',
    isOffered: (target) => target.isReady && target.file !== '' && target.file === file,
  }))
}

export interface FileMenuCommandsDeps {
  readonly get?: typeof getFileMenu
  readonly run?: typeof runFileMenuEntry
}

/**
 * The menu of one file, read again by `load` for the file named, each entry
 * with a handler that runs it and says the plugin is at work.
 */
export function useFileMenuCommands(deps: FileMenuCommandsDeps = {}) {
  const get = deps.get ?? getFileMenu
  const run = deps.run ?? runFileMenuEntry
  const menu = shallowRef<{ readonly file: string; readonly entries: readonly FileMenuEntry[] }>({
    file: '',
    entries: [],
  })
  let unregisterAll: readonly (() => void)[] = []

  const load = async (file: string): Promise<void> => {
    const entries = file ? await get(file) : []
    for (const unregister of unregisterAll) unregister()
    unregisterAll = entries.map((entry) =>
      registerCommand(getFileMenuId(entry), async (invocation, on) => {
        try {
          await run(invocation.file, entry)
          on.writeMessage(WORDS.textAsked(entry.pluginName), 'info')
        } catch (error) {
          on.writeMessage(formatErrorMessage(error), 'error')
        }
      }),
    )
    menu.value = { file, entries }
  }

  return {
    paletteCommands: computed(() =>
      createFileMenuPaletteCommands(menu.value.file, menu.value.entries),
    ),
    load,
  }
}

export type FileMenuCommands = ReturnType<typeof useFileMenuCommands>
