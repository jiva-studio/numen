/**
 * The shared commands of the application, as the window lists and runs them: the handlers the
 * plugins declare under `contributes.handlers`, each run by its plugin's frontend where it registered
 * the handler and by its backend otherwise.
 */
import { isSharedCommand, type CommandItem, type CommandNote } from '@numen/plugin-sdk'
import { formatCommandId } from '../commandId'
import { isCommandAvailable } from './commandWhen'
import { reportError, runCommand } from './events'
import type { HostRegistry } from './registry'
import type { CommandRun, HandlerDeclaration, HostDeps } from './types'

/** Joins the plugin id and the id of an item a handler offers, in the id of a shared command's item. */
const ITEM_SEPARATOR = ':'

/** Whether the full id is a shared command of the application. */

/** The items one handler offers of its own, and undefined for a handler that offers none. */
type HandlerItems = readonly CommandItem[] | undefined

/** What the handlers of the shared commands are read from and run through. */
interface HandlerParts {
  readonly deps: HostDeps
  readonly registry: HostRegistry
}

function listMatchingHandlers({ deps, registry }: HandlerParts, command: string, note: CommandNote) {
  return (deps.getHandlerDeclarations?.() ?? [])
    .filter((one) => one.command === command && !registry.disposed.has(one.pluginId))
    .filter((one) => (one.hasItems ? note.path !== '' : isCommandAvailable(one, note, deps)))
    .sort((a, b) => a.pluginId.localeCompare(b.pluginId))
}

const findEntry = ({ registry }: HandlerParts, handler: HandlerDeclaration) =>
  registry.handlers.value.get(formatCommandId(handler.pluginId, handler.command))

async function listOwnItems(
  parts: HandlerParts,
  handler: HandlerDeclaration,
  note: CommandNote,
): Promise<HandlerItems> {
  const entry = findEntry(parts, handler)
  try {
    if (entry?.reg.listItems) return await entry.reg.listItems(note)
    if (!entry && handler.hasItems) {
      return (
        (await parts.deps.listBackendHandlerItems?.(handler.pluginId, handler.command, note)) ?? []
      )
    }
  } catch (error) {
    reportError(parts.deps, handler.pluginId, error)
    return []
  }
  return undefined
}

/** One run of a shared command over the note. */
export interface SharedRun extends CommandRun {
  readonly note: CommandNote
}

async function runHandler(
  parts: HandlerParts,
  handler: HandlerDeclaration,
  run: SharedRun,
): Promise<void> {
  const { deps } = parts
  let entry = findEntry(parts, handler)
  if (!entry && deps.hasPluginFrontend?.(handler.pluginId) !== false) {
    await deps.waitForPlugin?.(handler.pluginId)
    entry = findEntry(parts, handler)
  }
  if (entry) {
    await runCommand(deps, entry, run)
    return
  }
  try {
    await deps.runBackendHandler?.({
      pluginId: handler.pluginId,
      command: handler.command,
      target: run.note,
      itemId: run.itemId,
      values: run.values,
    })
  } catch (error) {
    if (run.isAwaited) throw error
    reportError(deps, handler.pluginId, error)
  }
}

async function listHandlerItems(
  parts: HandlerParts,
  command: string,
  note: CommandNote,
): Promise<readonly CommandItem[]> {
  const handlers = listMatchingHandlers(parts, command, note)
  const own = await Promise.all(handlers.map((one) => listOwnItems(parts, one, note)))
  if (own.every((one) => one === undefined)) {
    return handlers.map((one) => ({ id: one.pluginId, title: one.title }))
  }
  return handlers.flatMap((one, index) =>
    (own[index] ?? [{ id: '', title: one.title }]).map((item) => ({
      id: item.id === '' ? one.pluginId : one.pluginId + ITEM_SEPARATOR + item.id,
      title: item.title,
      group: item.group ?? one.pluginId,
    })),
  )
}

async function runSharedCommand(
  parts: HandlerParts,
  command: string,
  run: SharedRun,
): Promise<boolean> {
  const { note, itemId, values } = run
  const handlers = listMatchingHandlers(parts, command, note)
  if (itemId === undefined) {
    const [only] = handlers
    if (handlers.length !== 1 || !only) return false
    await runHandler(parts, only, { note, values, isAwaited: run.isAwaited })
    return true
  }
  const separator = itemId.indexOf(ITEM_SEPARATOR)
  const pluginId = separator < 0 ? itemId : itemId.slice(0, separator)
  const handler = handlers.find((one) => one.pluginId === pluginId)
  if (!handler) return false
  const item = separator < 0 ? undefined : itemId.slice(separator + 1)
  await runHandler(parts, handler, { note, itemId: item, values, isAwaited: run.isAwaited })
  return true
}

export function createHandlerApi(deps: HostDeps, registry: HostRegistry) {
  const parts: HandlerParts = { deps, registry }
  return {
    isSharedCommand,
    listHandlerItems: (command: string, note: CommandNote) =>
      listHandlerItems(parts, command, note),
    runSharedCommand: (command: string, run: SharedRun) => runSharedCommand(parts, command, run),
  }
}
