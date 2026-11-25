/**
 * The commands the enabled plugins declare in their manifests, which the
 * plugins' backends run. The application checks a run against the declaration
 * before it is sent to the plugin.
 */
import {
  WIRE_ERROR_CODE,
  type PluginCommand,
  type PluginHandler,
  type PluginKeybinding,
  type PluginSlotEntry,
  type PluginSlotPoint,
  type RunPluginCommandResponse,
} from '@numen/core-client'
import {
  createWhenReader,
  type CommandAnswers,
  type CommandItem,
  type CommandWarns,
  type DeclaredSlotPoint,
  type HandlerDeclaration,
  type WhenReader,
} from '@numen/plugin-loader'
import type { WindowClients } from '@/shared/api/clients'
import { getRunPath } from '../lib/commandRun'

interface DeclaredArgument {
  readonly name: string
  readonly title: string
  readonly isRequired: boolean
}

/**
 * One entry of `contributes.slots` of an enabled plugin, as the core lists it. Exactly one of
 * command, view and submenu is set.
 */
export interface DeclaredSlot {
  readonly pluginId: string
  readonly slot: string
  readonly command?: string
  readonly view?: string
  readonly submenu?: string
  readonly title?: string
  readonly iconName?: string
  readonly group?: string
  readonly order?: number
  readonly when?: string
  readonly values?: Readonly<Record<string, string>>
}

/** One entry of `contributes.keybindings` of an enabled plugin, as the core lists it. */
export interface DeclaredKeybinding {
  readonly pluginId: string
  readonly key: string
  readonly command: string
  readonly values?: Readonly<Record<string, string>>
  readonly when?: string
}

/**
 * What the core lists of the enabled plugins: the commands they declare and where they place them.
 */
interface DeclaredList {
  readonly commands: readonly DeclaredCommand[]
  readonly slots: readonly DeclaredSlot[]
  readonly slotPoints?: readonly DeclaredSlotPoint[]
  readonly contentTypes?: Readonly<Record<string, string>>
  readonly handlers?: readonly HandlerDeclaration[]
  readonly keybindings?: readonly DeclaredKeybinding[]
}

/** One command a plugin declares. */
export interface DeclaredCommand {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  readonly when?: string
  readonly hasItems?: true
  readonly arguments: readonly DeclaredArgument[]
  readonly answers?: CommandAnswers
  readonly warns?: CommandWarns
}

/**
 * One run of a command: the file it is over, the values of its arguments by name, for a command
 * that offers items the id of the one picked, and whether a place named the file.
 */
export interface DeclaredCommandRun {
  readonly path: string
  readonly values: Readonly<Record<string, string>>
  readonly itemId?: string
  readonly isTargetNamed?: boolean
}

/**
 * The outcome of a run: the plugin's message and the file to open after the run,
 * either of them empty, or the reason the run failed.
 */
type DeclaredCommandOutcome =
  | { ok: true; value: { readonly message: string; readonly openedPath: string } }
  | { ok: false; error: string }

const readAnswers = (wire: PluginCommand['answers']): { answers: CommandAnswers } | object =>
  wire
    ? {
        answers: {
          keeps: wire.keeps,
          retainedNotice: wire.retainedNotice,
          action: wire.action,
          detail: wire.detail,
        },
      }
    : {}

const readWarns = (wire: PluginCommand['warns']): { warns: CommandWarns } | object =>
  wire ? { warns: { action: wire.action, detail: wire.detail, back: wire.back } } : {}

const readCommand = (wire: PluginCommand): DeclaredCommand => ({
  pluginId: wire.pluginId,
  id: wire.id,
  title: wire.title,
  ...(wire.icon ? { icon: wire.icon } : {}),
  ...(wire.when ? { when: wire.when } : {}),
  ...(wire.hasItems ? { hasItems: true as const } : {}),
  arguments: wire.arguments.map((one) => ({
    name: one.name,
    title: one.title,
    isRequired: one.isRequired,
  })),
  ...readAnswers(wire.answers),
  ...readWarns(wire.warns),
})

const PLACED_KINDS = ['command', 'view', 'submenu'] as const

const readPlacedEntry = (entry: PluginSlotEntry['entry']) => {
  const kind = PLACED_KINDS.find((one) => one === entry?.placed.case)
  return kind && entry?.placed.value ? { [kind]: entry.placed.value } : {}
}

const readArrangement = (entry: PluginSlotEntry['entry']) => ({
  ...(entry?.group ? { group: entry.group } : {}),
  ...(entry?.order === undefined ? {} : { order: entry.order }),
  ...(entry?.when ? { when: entry.when } : {}),
  ...(entry && Object.keys(entry.values).length > 0 ? { values: { ...entry.values } } : {}),
})

const readSlotEntry = ({ pluginId, entry }: PluginSlotEntry): DeclaredSlot => {
  return {
    pluginId,
    slot: entry?.slot ?? '',
    ...readPlacedEntry(entry),
    ...(entry?.title ? { title: entry.title } : {}),
    ...(entry?.icon ? { iconName: entry.icon } : {}),
    ...readArrangement(entry),
  }
}

const readSlotPoint = ({ pluginId, point }: PluginSlotPoint): DeclaredSlotPoint => ({
  pluginId,
  slot: point?.slot ?? '',
  title: point?.title ?? '',
  menu: point?.isMenu ?? false,
})

const readHandler = ({ pluginId, handler, hasItems }: PluginHandler): HandlerDeclaration => ({
  pluginId,
  command: handler?.command ?? '',
  title: handler?.title ?? '',
  ...(handler?.when ? { when: handler.when } : {}),
  hasItems,
})

const readKeybinding = ({ pluginId, keybinding }: PluginKeybinding): DeclaredKeybinding => ({
  pluginId,
  key: keybinding?.key ?? '',
  command: keybinding?.command ?? '',
  ...(keybinding?.values && Object.keys(keybinding.values).length > 0
    ? { values: { ...keybinding.values } }
    : {}),
  ...(keybinding?.when ? { when: keybinding.when } : {}),
})

/**
 * One command of a plugin whose items are listed: a command it declares, or its handler of a shared
 * command.
 */
export interface ItemListRequest {
  readonly pluginId: string
  readonly command: string
}

/** The items the core listed for one command, or why it could not list them. */
export interface ItemListAnswer extends ItemListRequest {
  readonly result:
    | { readonly ok: true; readonly value: readonly CommandItem[] }
    | { readonly ok: false; readonly error: string }
}

/** The code a command call or a list of items ends with. */
type CommandErrorValue = RunPluginCommandResponse['error']

const readOutcome = (answer: {
  readonly error: CommandErrorValue
  readonly message: string
  readonly openedPath?: string
}): DeclaredCommandOutcome =>
  answer.error === WIRE_ERROR_CODE.UNSPECIFIED
    ? { ok: true, value: { message: answer.message, openedPath: answer.openedPath ?? '' } }
    : { ok: false, error: answer.message }

const readItemList = (
  request: ItemListRequest,
  answer: {
    readonly error: CommandErrorValue
    readonly message: string
    readonly items: readonly {
      readonly id: string
      readonly title: string
      readonly detail: string
    }[]
  },
): ItemListAnswer => ({
  ...request,
  result:
    answer.error === WIRE_ERROR_CODE.UNSPECIFIED
      ? {
          ok: true,
          value: answer.items.map((item) => ({
            id: item.id,
            title: item.title,
            ...(item.detail ? { detail: item.detail } : {}),
          })),
        }
      : { ok: false, error: answer.message },
})

/**
 * The plugin id of the commands the application serves itself, such as Delete text: they are not
 * the commands of a plugin and go to the service of their subject.
 */
const CORE_PLUGIN = ''

/** The commands the plugins declare, listed and run over `pluginCommands`. */
export const createDeclaredCommandsApi = (
  pluginCommands: WindowClients['pluginCommands'],
  contributions: WindowClients['contributions'],
  textVersions: WindowClients['textVersions'],
) => {
  const runPluginCommand = async (
    pluginId: string,
    command: string,
    path: string,
    run: DeclaredCommandRun,
  ): Promise<DeclaredCommandOutcome> => {
    if (pluginId === CORE_PLUGIN) {
      return readOutcome(await textVersions.deleteVersion({ path, versionId: run.itemId ?? '' }))
    }
    return readOutcome(
      await pluginCommands.runPluginCommand({
        pluginId,
        command,
        path,
        arguments: { ...run.values },
        ...(run.itemId === undefined ? {} : { itemId: run.itemId }),
      }),
    )
  }

  return {
    /**
     * Every command the enabled plugins declare, in the order of the plugins' ids, their slot
     * entries and their keybindings.
     */
    listDeclaredCommands: async (): Promise<DeclaredList> => {
      const [listed, contributed, types] = await Promise.all([
        pluginCommands.listCommands({}),
        contributions.listContributions({}),
        contributions.listContentTypes({}),
      ])
      return {
        commands: listed.commands.map(readCommand),
        slots: contributed.slots.map(readSlotEntry),
        slotPoints: contributed.slotPoints.map(readSlotPoint),
        contentTypes: { ...types.contentTypeByExtension },
        handlers: contributed.handlers.map(readHandler),
        keybindings: contributed.keybindings.map(readKeybinding),
      }
    },

    /**
     * Runs the plugin's handler of the shared command in its backend, over the file, with the
     * values of the arguments the command declares and the item picked.
     */
    runDeclaredHandler: (
      handler: Pick<HandlerDeclaration, 'pluginId' | 'command'>,
      run: DeclaredCommandRun,
    ): Promise<DeclaredCommandOutcome> =>
      runPluginCommand(handler.pluginId, handler.command, run.path, run),

    /**
     * Runs the command in the backend of the plugin that declares it, over the file where its
     * condition reads the target, it offers items, or a place named the file. The reader reads the
     * condition; a window passes its one reader.
     */
    runDeclaredCommand: (
      command: DeclaredCommand,
      run: DeclaredCommandRun,
      whenReader: WhenReader = createWhenReader(),
    ): Promise<DeclaredCommandOutcome> =>
      runPluginCommand(command.pluginId, command.id, getRunPath(command, run, whenReader), run),

    /**
     * The items each of the commands offers over the file, as the core lists them: the commands of
     * plugins in one request, and the text versions the application lists itself.
     */
    listDeclaredItemLists: async (
      path: string,
      commands: readonly ItemListRequest[],
    ): Promise<readonly ItemListAnswer[]> => {
      const named = commands.filter((one) => one.pluginId !== CORE_PLUGIN)
      const [served, versions] = await Promise.all([
        named.length === 0
          ? undefined
          : pluginCommands.listCommandItems({
              path,
              commands: named.map((one) => ({ pluginId: one.pluginId, command: one.command })),
            }),
        commands.some((one) => one.pluginId === CORE_PLUGIN)
          ? textVersions.listVersionItems({ path })
          : undefined,
      ])
      const lists = [...(served?.lists ?? [])]
      return commands.map((one): ItemListAnswer => {
        if (one.pluginId === CORE_PLUGIN && versions !== undefined)
          return readItemList(one, versions)
        const list = lists.shift()
        return readItemList(
          { pluginId: list?.pluginId ?? one.pluginId, command: list?.command ?? one.command },
          list ?? { error: WIRE_ERROR_CODE.UNSPECIFIED, message: '', items: [] },
        )
      })
    },
  }
}

export type DeclaredCommandsApi = ReturnType<typeof createDeclaredCommandsApi>
