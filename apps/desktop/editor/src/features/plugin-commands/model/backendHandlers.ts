/**
 * The handlers of shared commands that run in their plugins' backends: the declarations the core
 * lists, the items the core lists for one, and a run of one that writes its message and opens the
 * file it returns, or rejects with a ReportedError of the plugin's message, which the caller of the run reports.
 */
import type { CommandItem, HandlerDeclaration, PluginHostDeps } from '@numen/plugin-loader'
import type { runDeclaredHandler } from '@/entities/plugin'
import { ReportedError } from '@/shared/api/errors'

export interface BackendHandlersDeps {
  readonly getHandlers: () => readonly HandlerDeclaration[] | undefined
  readonly list: (
    command: { readonly pluginId: string; readonly id: string },
    path: string,
  ) => Promise<readonly CommandItem[]>
  readonly run: typeof runDeclaredHandler
  readonly writeMessage: (text: string, severity: 'info') => void
  readonly openFile: (path: string) => Promise<void>
}

export function createBackendHandlers(
  deps: BackendHandlersDeps,
): Required<
  Pick<PluginHostDeps, 'getHandlerDeclarations' | 'listBackendHandlerItems' | 'runBackendHandler'>
> {
  return {
    getHandlerDeclarations: deps.getHandlers,
    listBackendHandlerItems: (pluginId, command, target) =>
      deps.list({ pluginId, id: command }, target.path),
    runBackendHandler: async ({ pluginId, command, target, itemId, values }) => {
      const outcome = await deps.run(
        { pluginId, command },
        {
          path: target.path,
          values: values ?? {},
          ...(itemId === undefined ? {} : { itemId }),
        },
      )
      if (!outcome.ok) throw new ReportedError(outcome.error)
      const { message, openedPath } = outcome.value
      if (message) deps.writeMessage(message, 'info')
      if (openedPath) await deps.openFile(openedPath)
    },
  }
}
