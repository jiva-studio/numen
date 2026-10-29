/**
 * The commands the enabled plugins declare, offered in the palette: one step
 * for each argument, and the run handed to the plugin's backend.
 */
import { computed, shallowRef } from 'vue'
import { registerCommand, type Command } from '@/features/command-palette'
import { getNamedIcon } from '@/shared/icons'
import {
  canRunOn,
  listDeclaredCommands,
  runDeclaredCommand,
  type DeclaredCommand,
} from '@/entities/plugin'

/** What the palette files a declared command under: the plugin and the command's own id. */
export const getPaletteId = (command: DeclaredCommand): string =>
  `plugin-command:${command.pluginId}:${command.id}`

/**
 * The palette's commands for the declared ones. A command over files is offered
 * over a file of one of its content types, and one over none over anything.
 */
export function createDeclaredPaletteCommands(
  declared: readonly DeclaredCommand[],
): readonly Command[] {
  return declared.map((one): Command => {
    const hasFiles = one.contentTypes.length > 0
    return {
      id: getPaletteId(one),
      text: one.title,
      group: hasFiles ? 'file' : 'window',
      ...(one.icon ? { icon: getNamedIcon(one.icon) } : {}),
      ...(one.arguments.length > 0
        ? {
            needs: 'arguments' as const,
            prompts: one.arguments.map(({ name, title, isRequired }) => ({
              name,
              title,
              isRequired,
            })),
          }
        : {}),
      isAvailable: (target) => target.isReady && (!hasFiles || canRunOn(one, target.file)),
    }
  })
}

export interface DeclaredCommandsDeps {
  readonly list?: typeof listDeclaredCommands
  readonly run?: typeof runDeclaredCommand
}

/**
 * The declared commands, reloaded by `load`, each registered with a handler that runs
 * it, writes the plugin message and opens the returned path.
 */
export function useDeclaredCommands(deps: DeclaredCommandsDeps = {}) {
  const list = deps.list ?? listDeclaredCommands
  const run = deps.run ?? runDeclaredCommand
  const declared = shallowRef<readonly DeclaredCommand[]>([])
  let unregisterAll: readonly (() => void)[] = []

  const load = async (): Promise<void> => {
    const commands = await list()
    for (const unregister of unregisterAll) unregister()
    unregisterAll = commands.map((one) =>
      registerCommand(getPaletteId(one), async (invocation, on) => {
        const outcome = await run(one, { path: invocation.file, values: invocation.values ?? {} })
        if (!outcome.ok) return on.writeMessage(outcome.error, 'error')
        const { message, openedPath } = outcome.value
        if (message) on.writeMessage(message, 'info')
        if (openedPath) on.notes.openFile(openedPath, '', 'current')
      }),
    )
    declared.value = commands
  }

  return {
    paletteCommands: computed(() => createDeclaredPaletteCommands(declared.value)),
    load,
  }
}

export type DeclaredCommands = ReturnType<typeof useDeclaredCommands>
