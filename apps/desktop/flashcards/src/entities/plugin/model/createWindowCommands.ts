/**
 * The commands the flashcards window gives its plugins: none. The window registers no command of its
 * own and binds none a plugin registers, so a place that draws a command shows none here.
 */
import type { CommandsHost } from '@numen/plugin-loader'

const NO_IDS: readonly string[] = []

export function createWindowCommands(): CommandsHost {
  return {
    getIds: () => NO_IDS,
    watch: (handler) => {
      handler(NO_IDS)
      return { dispose: () => undefined }
    },
    listItems: () => Promise.resolve([]),
    run: (command) => Promise.reject(new Error(`The flashcards window runs no command ${command}`)),
  }
}
