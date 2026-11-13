/**
 * The commands the plugins declare and what the host reads of the window before it exists: the
 * reports of the plugins, the slots, and the handlers of the backends.
 */
import type { Command } from '@/entities/command'
import { runDeclaredHandler } from '@/entities/plugin'
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
import { getWindowConfig } from '@/features/settings-commands'
import { formatError } from '@/shared/api/errors'
import type { CommandTarget } from '@/shared/model/command'
import type { MessageLog } from '@/shared/model/notices/messages'

/** What the plugins read of the window before the host exists. */
export interface PluginDeclarationsDeps {
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
  const { rest, log, itemLists, tabOpeners } = deps
  const catalog = createPluginCatalog(rest)
  const reports = createPluginReports(log, catalog.getName)
  const declared = createDeclaredState({
    getConfig: () => getWindowConfig(rest),
    paletteNotes: {
      read: createCommandNoteReader(itemLists),
      reportError: (error) => log.getWriter('command')(formatError(error), 'error'),
    },
  })
  const slots = createWindowSlots({
    declaredCommands: declared,
    facts: createWindowFacts({
      appCommands: deps.appCommands,
      getTarget: deps.getTarget,
      isPaletteOpen: deps.isPaletteOpen,
    }),
    writeIssue: reports.writeIssue,
  })
  const handlers = createBackendHandlers({
    getHandlers: () => declared.contributions.handlers.value,
    list: itemLists.listItems,
    run: runDeclaredHandler,
    writeMessage: log.getWriter('command'),
    openFile: (path) => tabOpeners.openFile(path, '', 'current').then(() => undefined),
  })
  const hostParts: readonly HostParts[] = [
    slots.hostDeps,
    handlers,
    {
      getCommandDeclarations: () => declared.declarations.value,
      reportCommandIssue: reports.reportCommandIssue,
      reportViewIssue: reports.reportViewIssue,
    },
  ]
  return { catalog, reports, declared, hostParts }
}
