/**
 * How the window starts and stops: the order its parts start in, and the order they stop in.
 */
import { runInBackground } from '@numen/ui'
import type { useVaults } from '@/entities/vault'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { createAppStart } from './createAppStart'
import { createAppStop } from './createAppStop'
import type { createLayoutStart } from './createLayoutStart'
import type { createPluginStart } from './createPluginStart'

export interface WindowBootstrapDeps {
  readonly vaults: Pick<ReturnType<typeof useVaults>, 'loadVaults'>
  readonly settings: {
    start(): Promise<void>
    close(): void
    readonly editorTokens: { start(): () => void }
  }
  readonly window: { start(): Promise<void>; close(): void }
  readonly editing: { readonly fileFlush: { start(): Promise<void> }; close(): void }
  readonly tabLinkSession: { start(): Promise<void>; close(): void }
  readonly layout: ReturnType<typeof createLayoutStart>
  readonly plugins: ReturnType<typeof createPluginStart>
  readonly writeMessage: MessageWriter
}

export function createWindowBootstrap(deps: WindowBootstrapDeps) {
  const { vaults, settings, window, editing, layout, plugins, writeMessage } = deps
  const reportError = (error: unknown) =>
    writeMessage(`The window did not start cleanly: ${formatError(error)}`, 'error')

  let stopEditorTokens = (): void => undefined
  const appStart = createAppStart({
    loadVaults: () => vaults.loadVaults(),
    startSettings: async () => {
      await settings.start()
      stopEditorTokens()
      stopEditorTokens = settings.editorTokens.start()
    },
    startLayout: layout.start,
    startPlugins: plugins.start,
    waitForInterfacePlugins: plugins.waitForInterface,
    startWindow: () => {
      runInBackground(deps.tabLinkSession.start(), (error) =>
        writeMessage(`The tab links stopped: ${formatError(error)}`, 'error'),
      )
      return window.start()
    },
    startEditing: () =>
      runInBackground(editing.fileFlush.start(), (error) =>
        writeMessage(`Saving on quit stopped: ${formatError(error)}`, 'error'),
      ),
    reportError,
  })
  createAppStop({
    closeWindow: () => {
      deps.tabLinkSession.close()
      window.close()
    },
    closeEditing: () => editing.close(),
    closeTabs: layout.close,
    closeSettings: () => {
      stopEditorTokens()
      settings.close()
    },
    closePlugins: plugins.close,
    reportError,
  })
  return appStart
}
