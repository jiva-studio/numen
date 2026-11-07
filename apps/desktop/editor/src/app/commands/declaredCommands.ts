/**
 * The commands the enabled plugins declare, offered in the palette: one step
 * for each required argument (an optional one is for a caller that is not a person),
 * and the run handed to the plugin's backend.
 */
import { computed, shallowRef, watch } from 'vue'
import {
  formatCommandId,
  type CommandDeclaration,
  type HandlerDeclaration,
  type WhenSettings,
} from '@numen/plugin-host'
import { registerCommand, type Command, type CommandTarget } from '@/features/command-palette'
import { getNamedIcon } from '@/shared/icons'
import {
  listDeclaredCommandItems,
  listDeclaredCommands,
  runDeclaredCommand,
  type DeclaredCommand,
  type DeclaredSlot,
} from '@/entities/plugin'
import {
  EMPTY_WHEN_CONTEXT,
  createWhenAvailability,
  getWhenGroup,
  type WindowWhenContext,
} from './whenContext'

/** What the palette files a command a plugin declares under: the plugin and the command's own id. */
const getPluginCommandId = (pluginId: string, commandId: string): string =>
  formatCommandId(pluginId, commandId)

/** What the palette files a declared command under. */
export const getPaletteId = (command: DeclaredCommand): string =>
  getPluginCommandId(command.pluginId, command.id)

/**
 * The palette's commands for the declared ones, each offered where its when condition holds over
 * what is in front, while the vault accepts operations, and grouped by what the condition reads.
 */
export function createDeclaredPaletteCommands(
  declared: readonly DeclaredCommand[],
  context: WindowWhenContext = EMPTY_WHEN_CONTEXT,
  listItems: typeof listDeclaredCommandItems = listDeclaredCommandItems,
): readonly Command[] {
  const isWhenAvailable = createWhenAvailability(context)
  return declared.map((one): Command => {
    const required = one.arguments.filter(({ isRequired }) => isRequired)
    return {
      id: getPaletteId(one),
      text: one.title,
      group: getWhenGroup(one.when),
      ...(one.icon ? { icon: getNamedIcon(one.icon) } : {}),
      ...(one.hasItems
        ? {
            input: 'items' as const,
            listItems: (target: CommandTarget) => listItems(one, target.file || target.path),
          }
        : {}),
      ...(required.length > 0
        ? {
            input: 'arguments' as const,
            prompts: required.map(({ name, title, isRequired }) => ({
              name,
              title,
              isRequired,
            })),
          }
        : {}),
      ...(one.when ? { when: one.when } : {}),
      // The versions a command with items runs over are the core's to list, over any file.
      isAvailable: (target) =>
        target.isReady &&
        (one.hasItems ? (target.file || target.path) !== '' : isWhenAvailable(one.when, target)),
    }
  })
}

/** Whether two lists the core answered hold the same entries in the same order. */
const isSameList = (
  current: readonly unknown[] | undefined,
  received: readonly unknown[],
): boolean => current !== undefined && JSON.stringify(current) === JSON.stringify(received)

export interface DeclaredCommandsDeps {
  readonly list?: typeof listDeclaredCommands
  readonly run?: typeof runDeclaredCommand
  /** Whether the plugin host lists the command of the full id: its frontend registered a handler, or is still loading. */
  readonly isListed?: (id: string) => boolean
  /** Whether the plugin declares a backend; a command is run in a backend only where it does. */
  readonly hasBackend?: (pluginId: string) => boolean
  /** The settings by their dotted keys, which the when conditions read as config. */
  readonly getConfig?: () => WhenSettings
}

/** The declaration the plugin host receives for a declared command. */
const createCommandDeclaration = (command: DeclaredCommand): CommandDeclaration => ({
  pluginId: command.pluginId,
  id: command.id,
  title: command.title,
  ...(command.icon ? { icon: command.icon } : {}),
  ...(command.when ? { when: command.when } : {}),
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
  const declaredSlots = shallowRef<readonly DeclaredSlot[] | undefined>(undefined)
  const declaredHandlers = shallowRef<readonly HandlerDeclaration[] | undefined>(undefined)
  const contentTypes = shallowRef<Readonly<Record<string, string>>>({})
  const whenContext: WindowWhenContext = {
    getContentTypes: () => contentTypes.value,
    getConfig: deps.getConfig ?? (() => ({})),
  }
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
          const outcome = await run(one, {
            path: invocation.file,
            values: invocation.values ?? {},
            ...(invocation.isTargetNamed ? { isTargetNamed: true } : {}),
            ...(one.hasItems ? { argument: invocation.name } : {}),
          })
          if (!outcome.ok) return on.writeMessage(outcome.error, 'error')
          const { message, openedPath } = outcome.value
          if (message) on.writeMessage(message, 'info')
          if (openedPath) on.notes.openFile(openedPath, '', 'current')
        }),
      )
    },
    { flush: 'sync' },
  )

  let onFirstList: () => void = () => undefined
  const firstList = new Promise<void>((resolve) => {
    onFirstList = resolve
  })

  const read = async (): Promise<void> => {
    newest += 1
    const mine = newest
    const commands = await list()
    if (mine !== newest) return
    // An answer equal to the lists held keeps them, so the commands are not registered again while
    // the palette lists them.
    if (!isSameList(declared.value, commands.commands)) declared.value = commands.commands
    if (!isSameList(declaredSlots.value, commands.slots)) declaredSlots.value = commands.slots
    const handlers = commands.handlers ?? []
    if (!isSameList(declaredHandlers.value, handlers)) declaredHandlers.value = handlers
    if (commands.contentTypes) contentTypes.value = commands.contentTypes
    onFirstList()
  }

  const load = (): Promise<void> =>
    read().catch((error: unknown) => {
      onFirstList()
      throw error
    })

  return {
    /** Settles once the lists of a read are held, or a read has failed. */
    waitForFirstList: (): Promise<void> => firstList,
    paletteCommands: computed(() =>
      createDeclaredPaletteCommands(backendCommands.value, whenContext),
    ),
    /** What the when conditions of the commands read besides their target. */
    whenContext,
    commands: computed(() => declared.value),
    slots: computed(() => declaredSlots.value),
    /** The handlers of shared commands the enabled plugins declare; undefined until the first list. */
    handlers: computed(() => declaredHandlers.value),
    declarations: computed(() => declared.value?.map(createCommandDeclaration)),
    load,
  }
}

export type DeclaredCommands = ReturnType<typeof useDeclaredCommands>
