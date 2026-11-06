/**
 * The shared commands of the application, which plugins handle in `contributes.handlers`. Over a
 * target a shared command has the handlers whose `when` holds there, offered as its items by plugin
 * id; a run with one of them runs it directly, and with several names the one to run.
 */
import { SHARED_COMMANDS, type CommandTarget } from '@numen/plugin-sdk'
import { parseWhen, type WhenCondition, type WhenValues } from '../when/when'

/** One handler a plugin declares in its manifest. */
export interface DeclaredHandler {
  readonly pluginId: string
  /** The full id of the shared command it handles. */
  readonly command: string
  readonly title: string
  readonly when?: string | undefined
}

/** A handler of a shared command as an item of the command: the plugin that handles it, and its title. */
export interface SharedCommandHandler {
  readonly id: string
  readonly title: string
}

export interface SharedCommandsDeps {
  /** The handlers of the enabled plugins; undefined until the first list has loaded. */
  readonly getHandlers: () => readonly DeclaredHandler[] | undefined
  /** The values the conditions of the handlers read over the target. */
  readonly getWhenValues: (target: CommandTarget | undefined) => WhenValues
  /** Runs the plugin's handler of the command over the target, and gives back what it returns. */
  readonly runHandler: (call: HandlerCall) => Promise<unknown>
}

/** One run of a handler: the plugin, the shared command, its target and the item it was given. */
export interface HandlerCall {
  readonly pluginId: string
  readonly command: string
  readonly target: CommandTarget | undefined
  readonly argument: unknown
}

/** Which handler a run picks, by its plugin id, and the item it gives the handler. */
export interface HandlerChoice {
  readonly pluginId?: string
  readonly argument?: unknown
}

const SHARED = new Set<string>(SHARED_COMMANDS.map((one) => one.id))

export function createSharedCommands(deps: SharedCommandsDeps) {
  const conditions = new Map<string, WhenCondition | undefined>()
  const isHolding = (handler: DeclaredHandler, values: WhenValues): boolean => {
    const text = handler.when ?? ''
    if (!conditions.has(text)) {
      const parsed = parseWhen(text)
      conditions.set(text, parsed.ok ? parsed.value : undefined)
    }
    return conditions.get(text)?.evaluate(values) ?? false
  }

  const listHandlers = (
    command: string,
    target: CommandTarget | undefined,
  ): SharedCommandHandler[] => {
    const values = deps.getWhenValues(target)
    return (deps.getHandlers() ?? [])
      .filter((one) => one.command === command && isHolding(one, values))
      .map((one) => ({ id: one.pluginId, title: one.title }))
  }

  return {
    isShared: (command: string): boolean => SHARED.has(command),
    listHandlers,
    /**
     * Runs the handler the plugin id names, or the one handler where it names none, and gives back
     * what the handler returns. `argument` goes to the handler, as an item it offers.
     */
    run: async (
      command: string,
      target: CommandTarget | undefined,
      choice: HandlerChoice = {},
    ): Promise<unknown> => {
      const { pluginId, argument } = choice
      const handlers = listHandlers(command, target)
      const where = target?.path ?? 'no target'
      if (pluginId === undefined && handlers.length !== 1) {
        throw new Error(
          `${command} has ${handlers.length} handlers over ${where}; the run names none of them`,
        )
      }
      const chosen = pluginId ?? handlers[0]!.id
      if (!handlers.some((one) => one.id === chosen)) {
        throw new Error(`${command} has no handler of ${chosen} over ${where}`)
      }
      return deps.runHandler({ pluginId: chosen, command, target, argument })
    },
  }
}

export type SharedCommands = ReturnType<typeof createSharedCommands>
