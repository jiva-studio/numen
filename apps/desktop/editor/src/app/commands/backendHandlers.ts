/**
 * The handlers of shared commands that run in their plugins' backends: the declarations the core
 * lists, the items the core lists for one, and a run of one that writes its message or its error and
 * opens the file it returns.
 */
import type { CommandItem, HandlerDeclaration, PluginHostDeps } from '@numen/plugin-host'
import type { runDeclaredHandler } from '@/entities/plugin'

export interface BackendHandlersDeps {
  /** The handlers the core lists; undefined until the first list. */
  readonly getHandlers: () => readonly HandlerDeclaration[] | undefined
  /** The items the command of the plugin offers over the file. */
  readonly list: (
    command: { readonly pluginId: string; readonly id: string },
    path: string,
  ) => Promise<readonly CommandItem[]>
  readonly run: typeof runDeclaredHandler
  readonly writeMessage: (text: string, severity: 'info' | 'error') => void
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
    runBackendHandler: async ({ pluginId, command, target, argument, values }) => {
      const outcome = await deps.run(
        { pluginId, command },
        {
          path: target.path,
          values: values ?? {},
          ...(argument === undefined ? {} : { argument }),
        },
      )
      if (!outcome.ok) return deps.writeMessage(outcome.error, 'error')
      const { message, openedPath } = outcome.value
      if (message) deps.writeMessage(message, 'info')
      if (openedPath) await deps.openFile(openedPath)
    },
  }
}
