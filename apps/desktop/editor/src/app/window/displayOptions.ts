import { formatError } from '@/shared/api/errors'
import type { VaultChangeEvent } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { CommandTarget } from '@/entities/command'
import type { MessageWriter } from '@/shared/notices/messages'
import type { PathRename } from '@/shared/paths'
import type { TextRange } from '@/shared/textRange'
import type { createCommands } from './createCommands'
import type { createNoteEditors } from '../tabs/useNoteEditors'
import type { useVaults } from './useVaults'
import type { WindowDisplay } from '@/features/window-display'
import type { useTabKinds } from './useTabKinds'

export interface DisplayWiring {
  editing: ReturnType<typeof createNoteEditors>
  getCommands: () => ReturnType<typeof createCommands>
  getKinds: () => ReturnType<typeof useTabKinds>
  tabOpeners: ReturnType<typeof fileOpeners>
  windowTabs: Pick<WindowTabs, 'groups'>
  getVaults: () => ReturnType<typeof useVaults>
  notifyPlugins: (event: VaultChangeEvent) => void
  writeMessage: MessageWriter
}

export function createDisplayOptions(deps: DisplayWiring) {
  const { editing, getCommands, getKinds, tabOpeners, windowTabs, getVaults, notifyPlugins } = deps
  const { writeMessage } = deps
  return {
    onVaultChanged: (
      paths: readonly string[],
      renamed: readonly PathRename[],
      versionPaths: readonly string[] = [],
    ) => {
      const changed = [...new Set([...paths, ...versionPaths])]
      // A change of the text versions alone changes no note.
      if (paths.length === 0 && renamed.length === 0 && versionPaths.length > 0) {
        notifyPlugins({ paths: changed, renamed })
        return
      }
      editing.notes.applyPathChanges(paths, renamed)
      // Every open tab on a moved or renamed file is navigated to the new path.
      for (const one of renamed) {
        runInBackground(Promise.resolve(getKinds().leavePath(one.from, one.to)), (error) =>
          writeMessage(formatError(error), 'error'),
        )
      }
      getCommands().commands.applyRenames(renamed)
      notifyPlugins({ paths: changed, renamed })
    },
    reportChange: editing.changes.reportChange,
    travelTo: (path: string) => getKinds().travel(path),
    openFileAt: (path: string, spans: readonly TextRange[]) =>
      runInBackground(tabOpeners.openFileAt(path, spans), (error) =>
        writeMessage(formatError(error), 'error'),
      ),
    activateTab: (id: string) => windowTabs.groups.handle.show(id),
    reload: () => getVaults().reload(),
  }
}

export function createTargetResolver(
  windowTabs: WindowTabs,
  getVaults: () => ReturnType<typeof useVaults>,
  window: WindowDisplay,
) {
  const getFront = () => {
    const front = windowTabs.groups.handle.front()
    const tab = front?.id ?? ''
    const on = front && windowTabs.tabs.getTab(tab)?.kind.getTarget?.(front.state)
    return { tab, kind: front?.kind ?? null, on }
  }
  return (): CommandTarget => {
    const { tab, kind, on } = getFront()
    const file = on?.file ?? ''
    return {
      tab,
      kind,
      path: on?.path ?? '',
      title: on?.title ?? '',
      file,
      vault: getVaults().activeVault.value,
      isReady: !window.health.failure.value,
    }
  }
}
