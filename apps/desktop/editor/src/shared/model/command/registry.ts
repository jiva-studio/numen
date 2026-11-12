import type { CommandHandler } from './deps'

/**
 * The handlers that carry out the commands of the features and the plugins, by command id. The call
 * that removes a handler leaves a handler registered since untouched.
 */
export interface CommandRegistry {
  readonly register: (id: string, handler: CommandHandler) => () => void
  readonly registerAll: (handlers: Readonly<Record<string, CommandHandler>>) => void
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
    registerAll: (table) => {
      for (const [id, handler] of Object.entries(table)) handlers.set(id, handler)
    },
    get: (id) => handlers.get(id),
  }
}
