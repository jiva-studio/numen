import type { CommandHandler } from './deps'

/** The handlers that carry out the commands of the features and the plugins, by command id. */
export interface CommandRegistry {
  /** Answers the call that removes the handler, which leaves a handler registered since untouched. */
  readonly register: (id: string, handler: CommandHandler) => () => void
  readonly get: (id: string) => CommandHandler | undefined
}

export function createCommandRegistry(): CommandRegistry {
  const handlers = new Map<string, CommandHandler>()
  return {
    register: (id, handler) => {
      handlers.set(id, handler)
      return () => {
        if (handlers.get(id) === handler) {
          handlers.delete(id)
        }
      }
    },
    get: (id) => handlers.get(id),
  }
}
