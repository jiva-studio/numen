/**
 * The connection between the plugin host and the commands of the window: what a plugin reads of the
 * commands, and the commands a plugin runs.
 */
import type { PluginHost } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { CommandDeps, CommandTarget } from '@/shared/model/command'
import { createInvocationRunner, createVaultOpener } from '@/features/command-palette'
import { createWindowCommands, getAppCommandIds, getListedCommandIds, type HostPort } from '@/features/plugin-host'
import type { createCommandRuns, createCommandSources } from './createCommands'

/** What the commands a plugin reads and runs are made from. */
export interface HostConnectionDeps {
  readonly hostPort: HostPort
  readonly pluginHost: Pick<PluginHost, 'commands'>
  readonly appCommands: readonly Command[]
  readonly getTarget: () => CommandTarget
  readonly items: ReturnType<typeof createCommandSources>['items']
  readonly runCommandToEnd: ReturnType<typeof createCommandRuns>['runCommandToEnd']
  readonly isPaletteOpen: () => boolean
  readonly commandDeps: CommandDeps
}

/** Connects the port the plugin host reads to the commands of the window, and returns them. */
export function createHostConnection(deps: HostConnectionDeps) {
  const { hostPort, items, getTarget, commandDeps } = deps
  const appIds = getAppCommandIds(deps.appCommands)
  const windowCommands = createWindowCommands({
    getAppIds: () => appIds,
    getPluginIds: () => getListedCommandIds(deps.pluginHost),
    runCommandToEnd: deps.runCommandToEnd,
    listItems: items.listCommandItems,
    runWithItem: items.runWithItem,
    getTarget,
  })
  hostPort.connect({
    commands: windowCommands,
    isPaletteOpen: deps.isPaletteOpen,
    runInvocation: createInvocationRunner(() => commandDeps),
    openVault: createVaultOpener(commandDeps, getTarget),
  })
  return windowCommands
}
