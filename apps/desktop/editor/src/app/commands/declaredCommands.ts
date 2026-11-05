/**
 * The commands the enabled plugins declare, offered in the palette: one step
 * for each required argument (an optional one is for a caller that is not a person),
 * and the run handed to the plugin's backend.
 */
import { computed, shallowRef, watch } from 'vue'
import { formatCommandId, type CommandDeclaration, type MenuPlacement } from '@numen/plugin-host'
import { registerCommand, type Command } from '@/features/command-palette'
import { getNamedIcon } from '@/shared/icons'
import {
  canRunOn,
  listDeclaredCommands,
  runDeclaredCommand,
  type DeclaredCommand,
  type DeclaredMenu,
} from '@/entities/plugin'

/** What the palette files a command a plugin declares under: the plugin and the command's own id. */
export const getPluginCommandId = (pluginId: string, commandId: string): string =>
  formatCommandId(pluginId, commandId)

/** What the palette files a declared command under. */
export const getPaletteId = (command: DeclaredCommand): string =>
  getPluginCommandId(command.pluginId, command.id)

/**
 * The palette's commands for the declared ones. A command over files is offered
 * over a file of one of its content types, and one over none over anything.
 */
export function createDeclaredPaletteCommands(
  declared: readonly DeclaredCommand[],
): readonly Command[] {
  return declared.map((one): Command => {
    const hasFiles = one.contentTypes.length > 0
    const required = one.arguments.filter(({ isRequired }) => isRequired)
    return {
      id: getPaletteId(one),
      text: one.title,
      group: hasFiles ? 'file' : 'window',
      ...(one.icon ? { icon: getNamedIcon(one.icon) } : {}),
      ...(required.length > 0
        ? {
            needs: 'arguments' as const,
            prompts: required.map(({ name, title, isRequired }) => ({
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
  /** Whether the plugin host lists the command of the full id: its frontend registered a handler, or is still loading. */
  readonly isListed?: (id: string) => boolean
  /** Whether the plugin declares a backend; a command is run in a backend only where it does. */
  readonly hasBackend?: (pluginId: string) => boolean
}

/** The declaration the plugin host receives for a declared command. */
const createCommandDeclaration = (command: DeclaredCommand): CommandDeclaration => ({
  pluginId: command.pluginId,
  id: command.id,
  title: command.title,
  ...(command.icon ? { icon: command.icon } : {}),
  ...(command.scope ? { scope: command.scope } : {}),
  ...(command.shortcut ? { shortcut: command.shortcut } : {}),
})

/**
 * The declared commands, reloaded by `load`, each registered with a handler that runs
 * it, writes the plugin message and opens the returned path.
 */
export function useDeclaredCommands(deps: DeclaredCommandsDeps = {}) {
  const list = deps.list ?? listDeclaredCommands
  const run = deps.run ?? runDeclaredCommand
  const isListed = deps.isListed ?? (() => false)
  const hasBackend = deps.hasBackend ?? (() => true)
  const declared = shallowRef<readonly DeclaredCommand[] | undefined>(undefined)
  const declaredMenus = shallowRef<readonly DeclaredMenu[] | undefined>(undefined)
  let newest = 0
  let unregisterAll: readonly (() => void)[] = []

  /**
   * The declared commands a backend runs and the palette lists: those the plugin host does not list,
   * because no frontend handler is registered for them and their frontend is not loading, and only
   * for a plugin that has a backend.
   */
  const backendCommands = computed(() =>
    (declared.value ?? []).filter(
      (one) => hasBackend(one.pluginId) && !isListed(getPaletteId(one)),
    ),
  )

  watch(
    backendCommands,
    (commands) => {
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
    },
    { flush: 'sync' },
  )

  const load = async (): Promise<void> => {
    newest += 1
    const mine = newest
    const commands = await list()
    if (mine !== newest) return
    declared.value = commands.commands
    declaredMenus.value = commands.menus
  }

  return {
    paletteCommands: computed(() => createDeclaredPaletteCommands(backendCommands.value)),
    commands: computed(() => declared.value),
    placements: computed((): readonly MenuPlacement[] | undefined => declaredMenus.value),
    declarations: computed(() => declared.value?.map(createCommandDeclaration)),
    load,
  }
}

export type DeclaredCommands = ReturnType<typeof useDeclaredCommands>
