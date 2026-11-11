/**
 * Where the commands over a note take the window: the plugin that shows notes, and the tabs.
 */
import type { Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { WindowTabs } from '@/entities/tab'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { NoteNavigator } from './commands'

/** What the navigator of the note commands reaches in the window. */
export interface NoteNavigatorDeps {
  readonly navigation: {
    readonly travel: (path: string) => Promise<void>
    readonly leavePath: (from: string, to: string) => Promise<void>
  }
  readonly opening: { readonly path: Readonly<Ref<string>> }
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
  readonly writeMessage: MessageWriter
}

export function createNoteNavigator(deps: NoteNavigatorDeps): NoteNavigator {
  const { navigation, opening, windowTabs, writeMessage } = deps
  return {
    travel: async (path) => {
      await navigation.travel(path)
    },
    leave: async (from, to) => {
      await navigation.leavePath(from, to)
    },
    getOpeningNote: () => opening.path.value,
    showTab: (kind, path) =>
      runInBackground(windowTabs.tabs.navigateTab(kind, path), (error) =>
        writeMessage(formatError(error), 'error'),
      ),
  }
}
