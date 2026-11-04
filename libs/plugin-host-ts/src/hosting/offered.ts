import { formatCommandId } from '../commandId'
import type { HostRegistry, RegisteredCommand } from './registry'
import type { CommandDeclaration, PluginHostDeps, PluginPaletteCommand } from './types'

/** A command the window offers: what its manifest declares and, where the plugin registered one, its handler. */
export interface OfferedCommand {
  /** The full id: the owner, a colon, and the id the manifest declares. */
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

/** What a listing of the offered commands returns. */
export interface OfferedList {
  readonly commands: OfferedCommand[]
  readonly issues: CommandIssue[]
}

/**
 * What becomes of a declared command with no handler. It is 'offered' while the frontend of its
 * plugin has not started to register. It is 'unbound' once the frontend has started: the host does
 * not offer it, and the window runs it in the backend of the plugin where it has one. A plugin with
 * no backend leaves the missing handler as an issue. It is 'left' to the window for a plugin with no frontend and for one that
 * failed to load.
 */
function getBeforeHandlerState(
  deps: PluginHostDeps,
  registry: HostRegistry,
  declaration: CommandDeclaration,
): 'offered' | 'unbound' | 'left' {
  if (deps.hasPluginFrontend?.(declaration.pluginId) === false) return 'left'
  if (deps.isPluginFailed?.(declaration.pluginId)) return 'left'
  return registry.activated.value.has(declaration.pluginId) ? 'unbound' : 'offered'
}

/**
 * The commands of the plugin frontends the window offers: those the manifests declare that have a
 * handler, and those of a frontend that has not started yet. A command with no handler of a plugin
 * that is not loading is left to the window, which runs it in the backend of the plugin. Nothing is
 * offered, and no issue is returned, until the first list of declarations has loaded. The listing
 * changes nothing and reports nothing.
 */
export function listOfferedCommands(deps: PluginHostDeps, registry: HostRegistry): OfferedList {
  const declarations = deps.getCommandDeclarations?.()
  if (declarations === undefined) return { commands: [], issues: [] }
  const commands: OfferedCommand[] = []
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
    if (state === 'offered') commands.push({ id, declaration })
    if (state === 'unbound' && deps.hasPluginBackend?.(declaration.pluginId) !== true) {
      addIssue({ pluginId: declaration.pluginId, id: declaration.id, kind: 'unbound' })
    }
  }
  for (const [id, entry] of registry.commands.value) {
    if (declaredIds.has(id)) continue
    addIssue({ pluginId: entry.pluginId, id: entry.reg.id, kind: 'undeclared' })
  }
  return { commands, issues: [...issues.values()] }
}

/** The palette's view of an offered command. */
export function createPaletteCommand({ id, declaration }: OfferedCommand): PluginPaletteCommand {
  return {
    id,
    title: declaration.title,
    ...(declaration.shortcut ? { shortcut: declaration.shortcut } : {}),
    ...(declaration.scope ? { scope: declaration.scope } : {}),
    ...(declaration.icon ? { icon: declaration.icon } : {}),
  }
}
