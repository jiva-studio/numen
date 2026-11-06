/**
 * The shared commands of the application, as the window lists and runs them: the handlers the
 * plugins declare under `contributes.handlers`, each run by its plugin's frontend where it registered
 * the handler and by its backend otherwise.
 */
import { SHARED_COMMANDS, type CommandItem, type CommandNote } from '@numen/plugin-sdk'
import { formatCommandId } from '../commandId'
import { isCommandAvailable } from './commandWhen'
import { reportError, runCommand } from './events'
import type { HostRegistry } from './registry'
import type { HandlerDeclaration, PluginHostDeps } from './types'

/** Joins the plugin id and the id of an item a handler offers, in the id of a shared command's item. */
const ITEM_SEPARATOR = ':'

/** Whether the full id is a shared command of the application. */
export const isSharedCommand = (id: string): boolean => SHARED_COMMANDS.some((one) => one.id === id)

/** The items one handler offers of its own, and undefined for a handler that offers none. */
type HandlerItems = readonly CommandItem[] | undefined

export function createHandlerApi(deps: PluginHostDeps, registry: HostRegistry) {
  /** The handlers of the command whose condition holds over the note, in the order of the plugins' ids. */
  const listMatching = (command: string, note: CommandNote): HandlerDeclaration[] =>
    (deps.getHandlerDeclarations?.() ?? [])
      .filter((one) => one.command === command && !registry.disposed.has(one.pluginId))
      .filter((one) => (one.hasItems ? note.path !== '' : isCommandAvailable(one, note, deps)))
      .sort((a, b) => a.pluginId.localeCompare(b.pluginId))

  const findEntry = (handler: HandlerDeclaration) =>
    registry.handlers.value.get(formatCommandId(handler.pluginId, handler.command))

  /** The items the handler offers of its own: those its frontend lists, or the core's for its backend. */
  const listOwnItems = async (
    handler: HandlerDeclaration,
    note: CommandNote,
  ): Promise<HandlerItems> => {
    const entry = findEntry(handler)
    try {
      if (entry?.reg.listItems) return await entry.reg.listItems(note)
      if (!entry && handler.hasItems) {
        return (await deps.listBackendHandlerItems?.(handler.pluginId, handler.command, note)) ?? []
      }
    } catch (error) {
      reportError(deps, handler.pluginId, error)
      return []
    }
    return undefined
  }

  /** Runs the handler over the note with the argument: in its frontend, or else in its backend. */
  const runHandler = async (handler: HandlerDeclaration, note: CommandNote, argument?: string) => {
    let entry = findEntry(handler)
    if (!entry && deps.hasPluginFrontend?.(handler.pluginId) !== false) {
      await deps.waitForPlugin?.(handler.pluginId)
      entry = findEntry(handler)
    }
    if (entry) {
      await runCommand(deps, entry, { note, argument })
      return
    }
    try {
      await deps.runBackendHandler?.(handler.pluginId, handler.command, note, argument)
    } catch (error) {
      reportError(deps, handler.pluginId, error)
    }
  }

  return {
    isSharedCommand,
    /**
     * The items a shared command offers over the note: one for each handler whose condition holds,
     * by its plugin id and title; or, where those handlers offer items of their own, theirs in one
     * list, each in the group it gives or else in the group of its plugin. A handler that offers none
     * of its own is then an item of its own group.
     */
    listHandlerItems: async (
      command: string,
      note: CommandNote,
    ): Promise<readonly CommandItem[]> => {
      const handlers = listMatching(command, note)
      const own = await Promise.all(handlers.map((one) => listOwnItems(one, note)))
      if (own.every((one) => one === undefined)) {
        return handlers.map((one) => ({ id: one.pluginId, title: one.title }))
      }
      return handlers.flatMap((one, at) =>
        (own[at] ?? [{ id: '', title: one.title } as CommandItem]).map((item) => ({
          id: item.id === '' ? one.pluginId : one.pluginId + ITEM_SEPARATOR + item.id,
          title: item.title,
          group: item.group ?? one.pluginId,
        })),
      )
    },
    /**
     * Runs a shared command over the note. The argument names the item picked: a plugin id runs that
     * plugin's handler, `<plugin id>:<item id>` runs it with the item. Without one the command runs
     * its one handler whose condition holds. Reports whether a handler ran.
     */
    runSharedCommand: async (
      command: string,
      note: CommandNote,
      argument?: string,
    ): Promise<boolean> => {
      const handlers = listMatching(command, note)
      if (argument === undefined) {
        if (handlers.length !== 1) return false
        await runHandler(handlers[0]!, note)
        return true
      }
      const separator = argument.indexOf(ITEM_SEPARATOR)
      const pluginId = separator < 0 ? argument : argument.slice(0, separator)
      const handler = handlers.find((one) => one.pluginId === pluginId)
      if (!handler) return false
      await runHandler(handler, note, separator < 0 ? undefined : argument.slice(separator + 1))
      return true
    },
  }
}
