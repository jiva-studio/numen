/**
 * The commands the enabled plugins declare, offered in the palette: one step
 * for each required argument (an optional one is for a caller that is not a person),
 * and the run handed to the plugin's backend.
 */
import type { CommandRegistry } from '@/shared/model/command'
import { computed, shallowRef, watch } from 'vue'
import {
  createWhenReader,
  formatCommandId,
  type CommandDeclaration,
  type CommandItem,
  type HandlerDeclaration,
  type WhenReader,
  type WhenSettings,
} from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { CommandInvocation, CommandTarget } from '@/shared/model/command'
import { getNamedIcon } from '@/shared/ui/icons'
import type { Writable } from '@/entities/tab'
import {
  listDeclaredItemLists,
  listDeclaredCommands,
  runDeclaredCommand,
  type DeclaredCommand,
  type DeclaredKeybinding,
  type DeclaredSlot,
} from '@/entities/plugin'
import {
  createEmptyWhenContext,
  createWhenAvailability,
  getWhenGroup,
  type WindowWhenContext,
} from '../lib/whenContext'
import type { Caught } from '@/shared/api/errors'

/** The items a command of a plugin offers over a file. */
type ListItems = (
  command: Pick<DeclaredCommand, 'pluginId' | 'id'>,
  path: string,
) => Promise<readonly CommandItem[]>

const listItemsOfOne: ListItems = async (command, path) => {
  const [answer] = await listDeclaredItemLists(path, [
    { pluginId: command.pluginId, command: command.id },
  ])
  if (!answer) throw new Error(`no list of the items of ${command.id}`)
  if (!answer.result.ok) throw new Error(answer.result.error)
  return answer.result.value
}

const getPluginCommandId = (pluginId: string, commandId: string): string =>
  formatCommandId(pluginId, commandId)

/** What the palette files a declared command under. */
export const getPaletteId = (command: DeclaredCommand): string =>
  getPluginCommandId(command.pluginId, command.id)

function createDeclaredPaletteCommands(
  declared: readonly DeclaredCommand[],
  context: WindowWhenContext = createEmptyWhenContext(),
  listItems: ListItems = listItemsOfOne,
): readonly Command[] {
  const isWhenAvailable = createWhenAvailability(context)
  return declared.map((one): Command => {
    const required = one.arguments.filter(({ isRequired }) => isRequired)
    return {
      id: getPaletteId(one),
      text: one.title,
      group: getWhenGroup(context.whenReader.getCondition(one.when)),
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

const isSameList = <T>(current: readonly T[] | undefined, received: readonly T[]): boolean =>
  current !== undefined && JSON.stringify(current) === JSON.stringify(received)

/** What the lists of declarations are read through. */
export interface DeclaredStateDeps {
  readonly list?: typeof listDeclaredCommands
  readonly getConfig?: () => WhenSettings
  readonly whenReader?: WhenReader
}

export interface DeclaredCommandsDeps extends DeclaredStateDeps {
  readonly state?: DeclaredState
  readonly registry?: Pick<CommandRegistry, 'register'>
  readonly run?: typeof runDeclaredCommand
  readonly isListed?: (id: string) => boolean
  readonly hasBackend?: (pluginId: string) => boolean
  readonly listItems?: ListItems
}

const createCommandDeclaration = (command: DeclaredCommand): CommandDeclaration => {
  const declaration: Writable<CommandDeclaration> = {
    pluginId: command.pluginId,
    id: command.id,
    title: command.title,
  }
  if (command.icon) declaration.icon = command.icon
  if (command.when) declaration.when = command.when
  return declaration
}

const toDeclaredRun = (one: DeclaredCommand, invocation: CommandInvocation) => ({
  path: invocation.target.file,
  values: invocation.values ?? {},
  ...(invocation.target.isTargetNamed ? { isTargetNamed: true } : {}),
  ...(one.hasItems ? { itemId: invocation.name } : {}),
})

/**
 * The lists the enabled plugins declare, reloaded by `load`: commands, slots, shared handlers and
 * keybindings, and what the conditions of the commands read.
 */
export function createDeclaredState(deps: DeclaredStateDeps = {}) {
  const list = deps.list ?? listDeclaredCommands
  const declared = shallowRef<readonly DeclaredCommand[] | undefined>(undefined)
  const declaredSlots = shallowRef<readonly DeclaredSlot[] | undefined>(undefined)
  const declaredHandlers = shallowRef<readonly HandlerDeclaration[] | undefined>(undefined)
  const declaredKeybindings = shallowRef<readonly DeclaredKeybinding[] | undefined>(undefined)
  const contentTypes = shallowRef<Readonly<Record<string, string>>>({})
  const whenContext: WindowWhenContext = {
    getContentTypes: () => contentTypes.value,
    getConfig: deps.getConfig ?? (() => ({})),
    whenReader: deps.whenReader ?? createWhenReader(),
  }
  let newest = 0
  const isFirstListSettled = shallowRef(false)

  const waitForFirstList = (): Promise<void> =>
    isFirstListSettled.value
      ? Promise.resolve()
      : new Promise((resolve) => {
          const stop = watch(
            isFirstListSettled,
            (isSettled) => {
              if (!isSettled) return
              stop()
              resolve()
            },
            { flush: 'sync' },
          )
        })

  const read = async (): Promise<void> => {
    newest += 1
    const mine = newest
    const commands = await list()
    if (mine !== newest) return
    // An answer equal to the lists held keeps them, so the commands are not registered again while
    // the palette lists them.
    const handlers = commands.handlers ?? []
    if (!isSameList(declared.value, commands.commands)) declared.value = commands.commands
    if (!isSameList(declaredSlots.value, commands.slots)) declaredSlots.value = commands.slots
    if (!isSameList(declaredHandlers.value, handlers)) declaredHandlers.value = handlers
    const keybindings = commands.keybindings ?? []
    if (!isSameList(declaredKeybindings.value, keybindings)) declaredKeybindings.value = keybindings
    if (commands.contentTypes) contentTypes.value = commands.contentTypes
    isFirstListSettled.value = true
  }

  return {
    /** Settles once the lists of a read are held, or a read has failed. */
    waitForFirstList,
    /** What the when conditions of the commands read besides their target. */
    whenContext,
    commands: computed(() => declared.value),
    slots: computed(() => declaredSlots.value),
    /** The handlers of shared commands and the keybindings the enabled plugins declare; undefined until the first list. */
    contributions: {
      handlers: computed(() => declaredHandlers.value),
      keybindings: computed(() => declaredKeybindings.value),
    },
    declarations: computed(() => declared.value?.map(createCommandDeclaration)),
    load: (): Promise<void> =>
      read().catch((error: Caught) => {
        isFirstListSettled.value = true
        throw error
      }),
  }
}

type DeclaredState = ReturnType<typeof createDeclaredState>

/**
 * The declared commands, reloaded by `load`, each registered with a handler that runs
 * it, writes the plugin message and opens the returned path.
 */
export function useDeclaredCommands(deps: DeclaredCommandsDeps = {}) {
  const state = deps.state ?? createDeclaredState(deps)
  const run = deps.run ?? runDeclaredCommand
  const isListed = deps.isListed ?? (() => false)
  const listItems = deps.listItems ?? listItemsOfOne
  const hasBackend = deps.hasBackend ?? (() => true)
  let unregisterAll: readonly (() => void)[] = []

  /**
   * The declared commands a backend runs and the palette lists: those the plugin host does not list,
   * because no frontend handler is registered for them and their frontend is not loading, and only
   * for a plugin that has a backend.
   */
  const backendCommands = computed(() =>
    (state.commands.value ?? []).filter(
      (one) => hasBackend(one.pluginId) && !isListed(getPaletteId(one)),
    ),
  )

  watch(
    backendCommands,
    (commands) => {
      for (const unregister of unregisterAll) unregister()
      const { registry } = deps
      if (!registry) return
      unregisterAll = commands.map((one) =>
        registry.register(getPaletteId(one), async (invocation, on) => {
          const outcome = await run(
            one,
            toDeclaredRun(one, invocation),
            state.whenContext.whenReader,
          )
          if (!outcome.ok) return on.writeMessage(outcome.error, 'error')
          const { message, openedPath } = outcome.value
          if (message) on.writeMessage(message, 'info')
          if (!openedPath) return
          on.notes.openFile(openedPath, '', 'current')
          return openedPath
        }),
      )
    },
    { flush: 'sync' },
  )

  return {
    waitForFirstList: state.waitForFirstList,
    paletteCommands: computed(() =>
      createDeclaredPaletteCommands(backendCommands.value, state.whenContext, listItems),
    ),
    whenContext: state.whenContext,
    commands: state.commands,
    slots: state.slots,
    contributions: state.contributions,
    declarations: state.declarations,
    load: state.load,
  }
}

export type DeclaredCommands = ReturnType<typeof useDeclaredCommands>
