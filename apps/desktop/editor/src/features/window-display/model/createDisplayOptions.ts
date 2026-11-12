import { formatError } from '@/shared/api/errors'
import type { VaultChangeEvent } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { FileOpeners, WindowTabs } from '@/entities/tab'
import type { Emitter } from '@/shared/lib/emitter'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { PathRename } from '@/shared/lib/paths'
import type { TextRange } from '@/shared/lib/textRange'

/** What a change of the vault reaches in the window. */
export interface DisplayWiring {
  editing: {
    notes: {
      applyPathChanges: (paths: readonly string[], renamed: readonly PathRename[]) => void
    }
  }
  navigation: {
    leavePath: (from: string, to: string) => Promise<void>
    travel: (path: string) => Promise<void> | void
  }
  renames: Emitter<readonly PathRename[]>
  tabOpeners: Pick<FileOpeners, 'openFileAt'>
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
    travelTo: navigation.travel,
    openFileAt: (path: string, spans: readonly TextRange[]) =>
      runInBackground(tabOpeners.openFileAt(path, spans), (error) =>
        writeMessage(formatError(error), 'error'),
      ),
    activateTab: (id: string) => windowTabs.groups.handle.show(id),
    reload: () => globalThis.location.reload(),
  }
}
