import { formatCommandId } from '../commandId'
import type { HostRegistry, RegisteredCommand } from './registry'
import type { CommandDeclaration, PluginHostDeps, PluginPaletteCommand } from './types'

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
  readonly kind: 'undeclared' | 'unbound'
}

/** The key an issue is reported once under: its plugin, id and kind. */
export const getIssueKey = (issue: CommandIssue): string =>
  `${issue.pluginId}:${issue.id}:${issue.kind}`

/** What a mismatch is reported as. */
export const getIssueMessage = (issue: CommandIssue): string =>
  issue.kind === 'undeclared'
    ? `the command ${issue.id} has a handler and no declaration in the manifest`
    : `the command ${issue.id} is declared in the manifest and has no handler`

/** What a listing of the listed commands returns. */
export interface AvailableList {
  readonly commands: AvailableCommand[]
  readonly issues: CommandIssue[]
}

function getBeforeHandlerState(
  deps: PluginHostDeps,
  registry: HostRegistry,
  declaration: CommandDeclaration,
): 'available' | 'unbound' | 'left' {
  if (deps.hasPluginFrontend?.(declaration.pluginId) === false) return 'left'
  if (deps.isPluginFailed?.(declaration.pluginId)) return 'left'
  if (!registry.activated.value.has(declaration.pluginId)) return 'available'
  if (registry.starting?.value.has(declaration.pluginId)) return 'available'
  return deps.hasPluginBackend?.(declaration.pluginId) === true ? 'left' : 'unbound'
}

/**
 * The commands of the plugin frontends the window lists: those the manifests declare that have a
 * handler, and those of a frontend that has not started yet. A command with no handler of a plugin
 * that is not loading is left to the window, which runs it in the backend of the plugin. Nothing is
 * listed, and no issue is returned, until the first list of declarations has loaded. The listing
 * changes nothing and reports nothing.
 */
export function listAvailableCommands(deps: PluginHostDeps, registry: HostRegistry): AvailableList {
  const declarations = deps.getCommandDeclarations?.()
  if (declarations === undefined) return { commands: [], issues: [] }
  const commands: AvailableCommand[] = []
  const issues = new Map<string, CommandIssue>()
  const addIssue = (issue: CommandIssue) => issues.set(getIssueKey(issue), issue)
  const declaredIds = new Set<string>()
  for (const declaration of declarations) {
    if (registry.disposed.has(declaration.pluginId)) continue
    const id = formatCommandId(declaration.pluginId, declaration.id)
    declaredIds.add(id)
    const entry = registry.commands.value.get(id)
    if (entry) {
      commands.push({ id, declaration, entry })
      continue
    }
    const state = getBeforeHandlerState(deps, registry, declaration)
    if (state === 'available') commands.push({ id, declaration })
    if (state === 'unbound') {
      addIssue({ pluginId: declaration.pluginId, id: declaration.id, kind: 'unbound' })
    }
  }
  for (const [id, entry] of registry.commands.value) {
    if (declaredIds.has(id)) continue
    addIssue({ pluginId: entry.pluginId, id: entry.reg.id, kind: 'undeclared' })
  }
  return { commands, issues: [...issues.values()] }
}

/** The palette's view of an listed command. */
export function createPaletteCommand({ id, declaration }: AvailableCommand): PluginPaletteCommand {
  return {
    id,
    title: declaration.title,
    ...(declaration.shortcut ? { shortcut: declaration.shortcut } : {}),
    ...(declaration.scope ? { scope: declaration.scope } : {}),
    ...(declaration.icon ? { icon: declaration.icon } : {}),
  }
}
