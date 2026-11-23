/**
 * The commands the plugins declare and what the host reads of the window before it exists: the
 * reports of the plugins, the slots, and the handlers of the backends.
 */
import type { Command } from '@/entities/command'
import type { DeclaredCommandsApi } from '@/entities/plugin'
import type { SettingsStore } from '@/entities/settings'
import type { FileOpeners } from '@/entities/tab'
import {
  createBackendHandlers,
  createCommandNoteReader,
  createDeclaredState,
  type CommandItemLists,
} from '@/features/plugin-commands'
import {
  createPluginCatalog,
  createPluginReports,
  createWindowFacts,
  createWindowSlots,
  type HostParts,
  type HostPort,
} from '@/features/plugin-host'
import { getSlotLayout, getWindowConfig } from '@/features/settings-commands'
import { formatError } from '@/shared/api/errors'
import type { CommandTarget } from '@/shared/model/command'
import type { MessageLog } from '@/shared/model/notices/messages'

/** What the plugins read of the window before the host exists. */
export interface PluginDeclarationsDeps {
  readonly api: DeclaredCommandsApi
  readonly rest: SettingsStore
  readonly log: MessageLog
  readonly isPaletteOpen: HostPort['isPaletteOpen']
  readonly getTarget: () => CommandTarget
  readonly appCommands: readonly Command[]
  readonly itemLists: CommandItemLists
  readonly tabOpeners: Pick<FileOpeners, 'openFile'>
}

/**
 * The catalog and the reports of the plugins, the commands they declare, and the parts of the host
 * that read them: the slots and the handlers of the backends.
 */
export function createPluginDeclarations(deps: PluginDeclarationsDeps) {
  const { api, rest, log, itemLists, tabOpeners } = deps
  const catalog = createPluginCatalog(rest)
  const pluginReports = createPluginReports(log, catalog.getName)
  const declarationState = createDeclaredState({
    list: api.listDeclaredCommands,
    getConfig: () => getWindowConfig(rest),
    paletteNotes: {
      read: createCommandNoteReader(itemLists),
      reportError: (error) => log.getWriter('command')(formatError(error), 'error'),
    },
  })
  const slots = createWindowSlots({
    declaredCommands: declarationState,
    facts: createWindowFacts({
      appCommands: deps.appCommands,
      getTarget: deps.getTarget,
      isPaletteOpen: deps.isPaletteOpen,
    }),
    getSlotLayout: (pluginId) => getSlotLayout(rest, pluginId),
    writeIssue: pluginReports.writeIssue,
  })
  const handlers = createBackendHandlers({
    getHandlers: () => declarationState.contributions.handlers.value,
    list: itemLists.listItems,
    run: api.runDeclaredHandler,
    writeMessage: log.getWriter('command'),
    openFile: (path) => tabOpeners.openFile(path, '', 'current').then(() => undefined),
  })
  const hostParts: readonly HostParts[] = [
    slots.hostDeps,
    { commands: handlers },
    {
      commands: {
        getCommandDeclarations: () => declarationState.declarations.value,
        reportCommandIssue: pluginReports.reportCommandIssue,
      },
      slots: { reportViewIssue: pluginReports.reportViewIssue },
    },
  ]
  return { catalog, reports: pluginReports, declared: declarationState, hostParts }
}
