import { formatCommandId } from '../commandId'
import type { HostRegistry, RegisteredCommand } from './registry'
import type { CommandDeclaration, HostPlugins, PluginHostDeps, PluginPaletteCommand } from './types'

/** A command the window lists: what its manifest declares and, where the plugin registered one, its handler. */
export interface AvailableCommand {
  readonly id: string
  readonly declaration: CommandDeclaration
  readonly entry?: RegisteredCommand
}

/** A mismatch between a declaration and a handler. */
export interface CommandIssue {
  readonly pluginId: string
  readonly id: string
  readonly kind: 'unbound'
}

/** The key an issue is reported once under: its plugin, id and kind. */
export const getIssueKey = (issue: CommandIssue): string =>
  `${issue.pluginId}:${issue.id}:${issue.kind}`

/** What a mismatch is reported as. */
export const getIssueMessage = (issue: CommandIssue): string =>
  `the command ${issue.id} is declared in the manifest and has no handler`

/** What a listing of the listed commands returns. */
export interface AvailableList {
  readonly commands: AvailableCommand[]
  readonly issues: CommandIssue[]
}

function getBeforeHandlerState(
  plugins: HostPlugins,
  registry: HostRegistry,
  declaration: CommandDeclaration,
): 'available' | 'unbound' | 'left' {
  if (plugins.hasPluginFrontend?.(declaration.pluginId) === false) return 'left'
  if (plugins.isPluginFailed?.(declaration.pluginId)) return 'left'
  if (!registry.activated.value.has(declaration.pluginId)) return 'available'
  if (registry.starting?.value.has(declaration.pluginId)) return 'available'
  return plugins.hasPluginBackend?.(declaration.pluginId) === true ? 'left' : 'unbound'
}

/**
 * The commands of the plugin frontends the window lists: those the manifests declare that have a
 * handler, and those of a frontend that has not started yet. A command with no handler of a plugin
 * that is not loading is left to the window, which runs it in the backend of the plugin. Nothing is
 * listed, and no issue is returned, until the first list of declarations has loaded. The listing
 * changes nothing and reports nothing.
 */
export function listAvailableCommands(
  deps: Pick<PluginHostDeps, 'plugins' | 'commands'>,
  registry: HostRegistry,
): AvailableList {
  const declarations = deps.commands?.getCommandDeclarations?.()
  if (declarations === undefined) return { commands: [], issues: [] }
  const commands: AvailableCommand[] = []
  const issues = new Map<string, CommandIssue>()
  const addIssue = (issue: CommandIssue) => issues.set(getIssueKey(issue), issue)
  for (const declaration of declarations) {
    if (registry.disposed.has(declaration.pluginId)) continue
    const id = formatCommandId(declaration.pluginId, declaration.id)
    const entry = registry.commands.value.get(id)
    if (entry) {
      commands.push({ id, declaration, entry })
      continue
    }
    const state = getBeforeHandlerState(deps.plugins ?? {}, registry, declaration)
    if (state === 'available') commands.push({ id, declaration })
    if (state === 'unbound') {
      addIssue({ pluginId: declaration.pluginId, id: declaration.id, kind: 'unbound' })
    }
  }
  return { commands, issues: [...issues.values()] }
}

/** The palette's view of a listed command. */
export function createPaletteCommand({ id, declaration }: AvailableCommand): PluginPaletteCommand {
  const command: { -readonly [K in keyof PluginPaletteCommand]: PluginPaletteCommand[K] } = {
    id,
    title: declaration.title,
  }
  if (declaration.when) command.when = declaration.when
  if (declaration.icon) command.icon = declaration.icon
  return command
}
