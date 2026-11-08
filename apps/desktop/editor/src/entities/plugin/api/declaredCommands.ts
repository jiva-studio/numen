/**
 * The commands the enabled plugins declare in their manifests, which the
 * plugins' backends run. The application checks a run against the declaration
 * before it is sent to the plugin.
 */
import { COMMAND_ERROR_CODE } from '@numen/core-client'
import { parseWhen, type CommandItem, type HandlerDeclaration } from '@numen/plugin-host'
import { pluginCommands } from '@/shared/clients'

interface WireCommand {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  readonly when: string
  readonly shortcut?: string
  readonly hasItems?: boolean
  readonly arguments: readonly {
    readonly name: string
    readonly title: string
    readonly isRequired: boolean
  }[]
}

interface WireSlotEntry {
  readonly pluginId: string
  readonly entry?:
    | {
        readonly slot: string
        readonly placed: { readonly case?: string | undefined; readonly value?: string | undefined }
        readonly title: string
        readonly icon: string
        readonly group: string
        readonly order?: number | undefined
        readonly when: string
      }
    | undefined
}

interface WireHandler {
  readonly pluginId: string
  readonly handler?:
    { readonly command: string; readonly title: string; readonly when: string } | undefined
  readonly hasItems: boolean
}

interface DeclaredArgument {
  readonly name: string
  readonly title: string
  readonly isRequired: boolean
}

/** One entry of `contributes.slots` of an enabled plugin, as the core lists it. Exactly one of command, view and submenu is set. */
export interface DeclaredSlot {
  readonly pluginId: string
  readonly slot: string
  readonly command?: string
  readonly view?: string
  readonly submenu?: string
  /** The title of a submenu. */
  readonly title?: string
  /** The Lucide name of the icon of a submenu. */
  readonly iconName?: string
  readonly group?: string
  readonly order?: number
  /** The condition over the target of the place under which the entry is drawn. */
  readonly when?: string
}

/** What the core lists of the enabled plugins: the commands they declare and where they place them. */
export interface DeclaredList {
  readonly commands: readonly DeclaredCommand[]
  readonly slots: readonly DeclaredSlot[]
  /** The content type of a file by its extension, lowercase and without the dot. */
  readonly contentTypes?: Readonly<Record<string, string>>
  /** The handlers of shared commands the enabled plugins declare. */
  readonly handlers?: readonly HandlerDeclaration[]
}

/** One command a plugin declares. */
export interface DeclaredCommand {
  readonly pluginId: string
  readonly id: string
  readonly title: string
  readonly icon?: string
  /** The CEL condition under which it is available, as the manifest declares it; absent is always. */
  readonly when?: string
  readonly shortcut?: string
  /** Set for a command that offers items, the text versions of a file it runs over. */
  readonly hasItems?: true
  readonly arguments: readonly DeclaredArgument[]
}

/**
 * One run of a command: the file it is over, the values of its arguments by name, for a command
 * that offers items the id of the one picked, and whether a place named the file.
 */
export interface DeclaredCommandRun {
  readonly path: string
  readonly values: Readonly<Record<string, string>>
  readonly argument?: string
  readonly isTargetNamed?: boolean
}

/**
 * The outcome of a run: the plugin's message and the file to open after the run,
 * either of them empty, or the reason the run failed.
 */
export type DeclaredCommandOutcome =
  | { ok: true; value: { readonly message: string; readonly openedPath: string } }
  | { ok: false; error: string }

const readCommand = (wire: WireCommand): DeclaredCommand => ({
  pluginId: wire.pluginId,
  id: wire.id,
  title: wire.title,
  ...(wire.icon ? { icon: wire.icon } : {}),
  ...(wire.when ? { when: wire.when } : {}),
  ...(wire.shortcut ? { shortcut: wire.shortcut } : {}),
  ...(wire.hasItems ? { hasItems: true as const } : {}),
  arguments: wire.arguments.map((one) => ({
    name: one.name,
    title: one.title,
    isRequired: one.isRequired,
  })),
})

const PLACED_KINDS = ['command', 'view', 'submenu'] as const

/** What an entry places, as the field of its kind. */
const readPlaced = (entry: WireSlotEntry['entry']) => {
  const kind = PLACED_KINDS.find((one) => one === entry?.placed.case)
  return kind && entry?.placed.value ? { [kind]: entry.placed.value } : {}
}

/** Where an entry goes among the others of its slot, and when it is drawn. */
const readArrangement = (entry: WireSlotEntry['entry']) => ({
  ...(entry?.group ? { group: entry.group } : {}),
  ...(entry?.order === undefined ? {} : { order: entry.order }),
  ...(entry?.when ? { when: entry.when } : {}),
})

const readSlotEntry = ({ pluginId, entry }: WireSlotEntry): DeclaredSlot => {
  return {
    pluginId,
    slot: entry?.slot ?? '',
    ...readPlaced(entry),
    ...(entry?.title ? { title: entry.title } : {}),
    ...(entry?.icon ? { iconName: entry.icon } : {}),
    ...readArrangement(entry),
  }
}

/** A handler of a shared command, with whether the core lists its items. */
const readHandler = ({ pluginId, handler, hasItems }: WireHandler): HandlerDeclaration => ({
  pluginId,
  command: handler?.command ?? '',
  title: handler?.title ?? '',
  ...(handler?.when ? { when: handler.when } : {}),
  hasItems,
})

/** Every command the enabled plugins declare, in the order of the plugins' ids, and their slot entries. */
export async function listDeclaredCommands(): Promise<DeclaredList> {
  const answer = await pluginCommands.listCommands({})
  return {
    commands: answer.commands.map(readCommand),
    slots: answer.slots.map(readSlotEntry),
    contentTypes: { ...answer.contentTypeByExtension },
    handlers: (answer.handlers as readonly WireHandler[] | undefined)?.map(readHandler) ?? [],
  }
}

/**
 * Runs the plugin's handler of the shared command in its backend, over the file, with the values of
 * the arguments the command declares and the item picked.
 */
export async function runDeclaredHandler(
  handler: Pick<HandlerDeclaration, 'pluginId' | 'command'>,
  run: DeclaredCommandRun,
): Promise<DeclaredCommandOutcome> {
  const answer = await pluginCommands.runPluginCommand({
    pluginId: handler.pluginId,
    command: handler.command,
    path: run.path,
    arguments: { ...run.values },
    ...(run.argument === undefined ? {} : { argument: run.argument }),
  })
  if (answer.error === COMMAND_ERROR_CODE.UNSPECIFIED) {
    return { ok: true, value: { message: answer.message, openedPath: answer.openedPath } }
  }
  return { ok: false, error: answer.message }
}

/**
 * Runs the command in the backend of the plugin that declares it, over the file where its condition
 * reads the target, it offers items, or a place named the file.
 */
export async function runDeclaredCommand(
  command: DeclaredCommand,
  run: DeclaredCommandRun,
): Promise<DeclaredCommandOutcome> {
  const answer = await pluginCommands.runPluginCommand({
    pluginId: command.pluginId,
    command: command.id,
    path: isOverTarget(command) || command.hasItems || run.isTargetNamed ? run.path : '',
    arguments: { ...run.values },
    ...(run.argument === undefined ? {} : { argument: run.argument }),
  })
  if (answer.error === COMMAND_ERROR_CODE.UNSPECIFIED) {
    return { ok: true, value: { message: answer.message, openedPath: answer.openedPath } }
  }
  return { ok: false, error: answer.message }
}

/** The items the command offers over the file, as the core lists them. */
export async function listDeclaredCommandItems(
  command: Pick<DeclaredCommand, 'pluginId' | 'id'>,
  path: string,
): Promise<readonly CommandItem[]> {
  const answer = await pluginCommands.listCommandItems({
    path,
    commands: [{ pluginId: command.pluginId, command: command.id }],
  })
  const list = answer.lists[0]
  if (!list) throw new Error(`no list of the items of ${command.id}`)
  if (list.error !== COMMAND_ERROR_CODE.UNSPECIFIED) throw new Error(list.message)
  return list.items.map((one) => ({ id: one.id, title: one.title }))
}

/** Whether the condition of the command reads its target, so a run of it names the path it is over. */
export function isOverTarget(command: Pick<DeclaredCommand, 'when'>): boolean {
  const parsed = parseWhen(command.when ?? '')
  return parsed.ok && parsed.value.names.some((name) => name.startsWith('target.'))
}
