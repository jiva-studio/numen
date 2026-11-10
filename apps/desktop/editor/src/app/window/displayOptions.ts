import { formatError } from '@/shared/api/errors'
import type { VaultChangeEvent } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { createTabNavigation, fileOpeners, WindowTabs } from '@/entities/tab'
import type { Emitter } from '@/shared/lib/emitter'
import type { CommandTarget } from '@/entities/command'
import type { MessageWriter } from '@/shared/notices/messages'
import type { PathRename } from '@/shared/paths'
import type { TextRange } from '@/shared/textRange'
import type { createNoteEditors } from '../tabs/createNoteEditors'
import type { useVaults } from '@/entities/vault'
import type { WindowDisplay } from '@/features/window-display'

export interface DisplayWiring {
  editing: ReturnType<typeof createNoteEditors>
  navigation: ReturnType<typeof createTabNavigation>
  /** Receives the renames of a change of the vault, for whatever keeps paths of its own. */
  renames: Emitter<readonly PathRename[]>
  tabOpeners: ReturnType<typeof fileOpeners>
  windowTabs: Pick<WindowTabs, 'groups'>
  notifyPlugins: (event: VaultChangeEvent) => void
  writeMessage: MessageWriter
}

export function createDisplayOptions(deps: DisplayWiring) {
  const { editing, navigation, renames, tabOpeners, windowTabs, notifyPlugins } = deps
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
        runInBackground(navigation.leavePath(one.from, one.to), (error) =>
          writeMessage(formatError(error), 'error'),
        )
      }
      renames.emit(renamed)
      notifyPlugins({ paths: changed, renamed })
    },
    reportChange: editing.changes.reportChange,
    travelTo: navigation.travel,
    openFileAt: (path: string, spans: readonly TextRange[]) =>
      runInBackground(tabOpeners.openFileAt(path, spans), (error) =>
        writeMessage(formatError(error), 'error'),
      ),
    activateTab: (id: string) => windowTabs.groups.handle.show(id),
    reload: () => globalThis.location.reload(),
  }
}

export function createTargetResolver(
  windowTabs: WindowTabs,
  vaults: Pick<ReturnType<typeof useVaults>, 'activeVault'>,
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
      vault: vaults.activeVault.value,
      isReady: !window.health.failure.value,
    }
  }
}
