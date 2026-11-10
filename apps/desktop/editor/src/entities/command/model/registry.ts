import type { CommandHandler } from './deps'

const registry = new Map<string, CommandHandler>()

/** The handlers that carry out the commands of the features and the plugins, by command id. */
export const commands = {
  /** Answers the call that removes the handler, which leaves a handler registered since untouched. */
  registerCommand: (id: string, handler: CommandHandler): (() => void) => {
    registry.set(id, handler)
    return () => {
      if (registry.get(id) === handler) {
        registry.delete(id)
      }
    }
  },
  get(id: string): CommandHandler | undefined {
    return registry.get(id)
  },
  clear(): void {
    registry.clear()
  },
}

export const registerCommand = commands.registerCommand
