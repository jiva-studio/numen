import { formatError } from '@/shared/api/errors'
import { runInBackground } from '@numen/ui'
import type { FileOpeners, WindowTabs } from '@/entities/tab'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { TextRange } from '@/shared/lib/textRange'

/** What a request from outside the window to show a note or a tab reaches in the window. */
export interface FocusDeps {
  navigation: {
    travel: (path: string) => Promise<void> | void
  }
  tabOpeners: Pick<FileOpeners, 'openFileAt'>
  windowTabs: Pick<WindowTabs, 'groups'>
  writeMessage: MessageWriter
}

/** Shows the note, the spans of a file or the tab that a request from outside the window names. */
export function createFocusHandlers(deps: FocusDeps) {
  const { navigation, tabOpeners, windowTabs, writeMessage } = deps
  return {
    travelTo: navigation.travel,
    openFileAt: (path: string, spans: readonly TextRange[]) =>
      runInBackground(tabOpeners.openFileAt(path, spans), (error) =>
        writeMessage(formatError(error), 'error'),
      ),
    activateTab: (id: string) => windowTabs.groups.handle.show(id),
  }
}
