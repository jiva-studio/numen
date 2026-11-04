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

interface CommandIssue {
  readonly pluginId: string
  readonly id: string
  readonly kind: 'undeclared' | 'unbound'
}

/** Reports a mismatch between a declaration and a handler once for each plugin, id and kind. */
function reportIssue(deps: PluginHostDeps, registry: HostRegistry, issue: CommandIssue): void {
  const key = `${issue.pluginId}:${issue.id}:${issue.kind}`
  if (registry.reportedIssues.has(key)) return
  registry.reportedIssues.add(key)
  deps.reportCommandIssue?.(
    issue.pluginId,
    issue.kind === 'undeclared'
      ? `the command ${issue.id} has a handler and no declaration in the manifest`
      : `the command ${issue.id} is declared in the manifest and has no handler`,
  )
}

/**
 * Whether a declared command with no handler is offered anyway: its plugin has a frontend that has
 * not started to register yet. A plugin that failed to load is not offered. Once the frontend has
 * started, the command is not offered by the host and the missing handler is reported; the window
 * runs the command in the backend of the plugin where it has one. A plugin with no frontend has
 * nothing to wait for, and its commands are left to the window.
 */
function isOfferedBeforeItsHandler(
  deps: PluginHostDeps,
  registry: HostRegistry,
  declaration: CommandDeclaration,
): boolean {
  if (deps.hasPluginFrontend?.(declaration.pluginId) === false) return false
  if (deps.isPluginFailed?.(declaration.pluginId)) return false
  if (!registry.activated.value.has(declaration.pluginId)) return true
  reportIssue(deps, registry, {
    pluginId: declaration.pluginId,
    id: declaration.id,
    kind: 'unbound',
  })
  return false
}

/**
 * The commands of the plugin frontends the window offers: those the manifests declare that have a
 * handler, and those of a frontend that has not started yet. A command with no handler of a plugin
 * that is not loading is left to the window, which runs it in the backend of the plugin. Nothing is
 * offered, and nothing is reported, until the first list of declarations has loaded.
 */
export function listOfferedCommands(
  deps: PluginHostDeps,
  registry: HostRegistry,
): OfferedCommand[] {
  const declarations = deps.getCommandDeclarations?.()
  if (declarations === undefined) return []
  const commands: OfferedCommand[] = []
  const declaredIds = new Set<string>()
  for (const declaration of declarations) {
    if (registry.disposed.has(declaration.pluginId)) continue
    const id = formatCommandId(declaration.pluginId, declaration.id)
    declaredIds.add(id)
    const entry = registry.commands.value.get(id)
    if (entry) commands.push({ id, declaration, entry })
    else if (isOfferedBeforeItsHandler(deps, registry, declaration))
      commands.push({ id, declaration })
  }
  for (const [id, entry] of registry.commands.value) {
    if (declaredIds.has(id)) continue
    reportIssue(deps, registry, { pluginId: entry.pluginId, id: entry.reg.id, kind: 'undeclared' })
  }
  return commands
}

/** The palette's view of an offered command. */
export function createPaletteCommand({ id, declaration }: OfferedCommand): PluginPaletteCommand {
  return {
    id,
    title: declaration.title,
    ...(declaration.shortcut ? { shortcut: declaration.shortcut } : {}),
    ...(declaration.scope ? { scope: declaration.scope } : {}),
    ...(declaration.icon ? { icon: declaration.icon } : {}),
    ...(declaration.isNoteDestination ? { isNoteDestination: true } : {}),
  }
}
