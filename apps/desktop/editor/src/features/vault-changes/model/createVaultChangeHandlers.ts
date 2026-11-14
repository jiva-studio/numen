import { formatError } from '@/shared/api/errors'
import type { VaultChangeEvent } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { Emitter } from '@/shared/lib/emitter'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { PathRename } from '@/shared/lib/paths'

/** What a change of the vault reaches in the window. */
export interface VaultChangeDeps {
  editing: {
    notes: {
      applyPathChanges: (paths: readonly string[], renamed: readonly PathRename[]) => void
    }
  }
  navigation: {
    leavePath: (from: string, to: string) => Promise<void>
  }
  renames: Emitter<readonly PathRename[]>
  notifyPlugins: (event: VaultChangeEvent) => void
  writeMessage: MessageWriter
}

/**
 * Applies a change of the vault to the open notes, the tabs and the plugins, and reloads the page
 * once another vault is under the window.
 */
export function createVaultChangeHandlers(deps: VaultChangeDeps) {
  const { editing, navigation, renames, notifyPlugins, writeMessage } = deps
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
    reload: () => globalThis.location.reload(),
  }
}
