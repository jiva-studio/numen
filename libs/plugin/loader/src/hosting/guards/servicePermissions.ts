/**
 * What each service of the plugin context and of the host services needs the plugin to hold, in
 * one table. A guarded service is built from the table, not from the service: a member the table
 * does not list is not on the guarded service, and the table's type lists every member of the SDK
 * types, so a member added to the SDK fails the typecheck until it states its permission or why it
 * needs none.
 *
 * The table catches mistakes and states what a plugin declares; it is not a security boundary.
 * Plugins are trusted code that runs in the window realm, and a hostile one reaches the core
 * endpoints, `fetch` and `import` without this table.
 */
import type {
  HealthHost,
  EditorExtensionHost,
  PluginEvents,
  PluginIdentity,
  PluginLicenses,
  PluginRegistrations,
  PluginSettings,
  PluginTabs,
  PluginUi,
  PluginVault,
  PluginWork,
} from '@numen/plugin-sdk'
import type { Failure } from '@numen/result'

/** What a member of a service holds: the guard passes members on without knowing their type. */
export type Value = object | string | number | bigint | boolean | symbol | null | undefined | void

export const READ = 'vault:read'
export const WRITE = 'vault:write'
const MANAGE = 'vault:manage'
/** Consent to open external links; it does not restrict the requests a plugin makes itself, as plugins are trusted code. */
const LINK_OPEN = 'link:open'
/** Reading which plugins run; nothing of a vault. */
const PLUGINS_READ = 'plugins:read'
/** Reading the structured application logs of the installation. */
const LOGS_READ = 'logs:read'
/** A model call uses the connections the plugin names in its own settings; the person granted them there. */
const CONNECTION_GRANT = `the rows of the plugin's own settings are the grant`

/**
 * How a refusal of a rule reaches the caller: a sync member throws, an async member rejects, a
 * result member answers `{ ok: false, error: { error: 'permissionDenied', message } }`, and a rule
 * that lists no permission cannot refuse, so it fits any.
 */
type Mode = 'sync' | 'async' | 'result' | 'none'

/** What a member needs: every permission listed, and whether a refusal throws or rejects. */
export class Rule<M extends Mode = Mode> {
  readonly reason: string
  readonly isReadOnly: (args: readonly Value[]) => boolean

  constructor(
    readonly permissions: readonly string[],
    readonly mode: M,
    details: { readonly reason: string; readonly isReadOnly?: (args: readonly Value[]) => boolean },
  ) {
    this.reason = details.reason
    this.isReadOnly = details.isReadOnly ?? (() => false)
  }

  get isAsync(): boolean {
    return this.mode === 'async'
  }

  get isResult(): boolean {
    return this.mode === 'result'
  }
}

function requirePermissions(...permissions: string[]): Rule<'sync'> {
  return new Rule(permissions, 'sync', { reason: '' })
}

function requirePermissionsAsync(...permissions: string[]): Rule<'async'> {
  return new Rule(permissions, 'async', { reason: '' })
}

function requirePermissionsResult(...permissions: string[]): Rule<'result'> {
  return new Rule(permissions, 'result', { reason: '' })
}

function requireNothing(reason: string): Rule<'none'> {
  return new Rule([], 'none', { reason })
}

function explain<M extends Mode>(rule: Rule<M>, reason: string): Rule<M> {
  return new Rule(rule.permissions, rule.mode, { reason, isReadOnly: rule.isReadOnly })
}

/** The commands that move a view and change nothing, so `vault:read` is enough to run them. */
const READ_ONLY_COMMANDS: ReadonlySet<Value> = new Set(['app:file.reveal', 'app:settings.open'])

function readOnlyFor<M extends Mode>(
  rule: Rule<M>,
  isReadOnly: (args: readonly Value[]) => boolean,
): Rule<M> {
  return new Rule(rule.permissions, rule.mode, { reason: rule.reason, isReadOnly })
}

type Leaf =
  | ((...args: never[]) => Value)
  | readonly Value[]
  | { readonly value: Value }
  | string
  | number
  | boolean

/** The failure of a Result, and never for any other type. */
type FailureOf<T> = T extends { readonly ok: false; readonly error: infer E } ? E : never

/** The failure a refusal answers with where the member answers a Result. */
type PermissionDenied = Failure<'permissionDenied'>

/**
 * The rule a member can have, from its type: a member that answers a promise of a Result whose failure
 * can be `permissionDenied` takes a result rule, so a refusal is an answer; any other member that
 * returns a promise (or may) takes an async rule, so a refusal rejects; any other member takes a sync
 * rule, so a refusal throws.
 */
type RuleOf<M> = M extends (...args: never[]) => infer R
  ? [Extract<R, PromiseLike<Value>>] extends [never]
    ? Rule<'sync' | 'none'>
    : [PermissionDenied] extends [FailureOf<Awaited<R>>]
      ? Rule<'result' | 'none'>
      : Rule<'async' | 'none'>
  : Rule<'sync' | 'none'>

/** The rules of a service: one for each member, nested for a member that is a service itself. */
export type RulesOf<S> = {
  readonly [K in keyof S]-?: NonNullable<S[K]> extends Leaf
    ? RuleOf<NonNullable<S[K]>>
    : RulesOf<NonNullable<S[K]>>
}

/** The parts of the plugin context and of the host services that offer services. */
export interface GuardedParts {
  readonly identity: Pick<PluginIdentity, 'openExternal'>
  readonly register: PluginRegistrations
  readonly settings: PluginSettings
  readonly licenses: PluginLicenses
  readonly events: PluginEvents
  readonly editor: EditorExtensionHost
  readonly tabs: PluginTabs
  readonly vault: PluginVault
  readonly ui: Omit<PluginUi, 'mountView'>
  readonly work: PluginWork
  readonly health: HealthHost
}

const EDITOR_LIST = explain(
  requirePermissions(WRITE),
  'the extensions are code other plugins run in the editor, so the plugin that receives them can change the documents too',
)

export const SERVICE_RULES: RulesOf<GuardedParts> = {
  identity: {
    openExternal: requirePermissionsResult(LINK_OPEN),
  },
  register: {
    reader: requireNothing('a reader names the extensions the manifest of the plugin declares'),
    settingsView: requireNothing('the view draws the settings of the plugin itself'),
    slotView: requireNothing('the slots are the ones the manifest of the plugin places it in'),
    tabKind: requirePermissions(READ),
    command: requireNothing('a command runs where the user or a plugin invokes it'),
    view: requireNothing(
      'the manifest must list the kind and the plugin hold what the kind hands its views: checked in viewPermissions',
    ),
    dropHandler: explain(
      requirePermissions(READ),
      'dropped data is user content, and the handler receives it and the folder it was dropped into',
    ),
    editorExtension: explain(
      requirePermissions(WRITE),
      'it runs in the editor of every open note and can change the document',
    ),
  },
  settings: {
    get: requireNothing('it reads the settings of the plugin itself'),
    onChange: requireNothing('it listens to the settings of the plugin itself'),
    getEditor: requireNothing('it reads the editor settings of the application, read-only'),
    onEditorChange: requireNothing('it listens to the editor settings of the application'),
  },
  licenses: {
    has: requireNothing('it reads the state of a license the manifest of the plugin declares'),
    getKey: requireNothing('it reads the key of a license the manifest of the plugin declares'),
    onChange: requireNothing(
      'it listens to the state of a license the manifest of the plugin declares',
    ),
  },
  events: {
    onVaultChange: requirePermissions(READ),
    onFileSelect: requirePermissions(READ),
    emitVaultChange: requirePermissions(WRITE),
    emitFileSelect: explain(
      requirePermissions(WRITE),
      'the selection reaches every plugin that follows it, which then acts on the path: it is an event the plugin makes, as a vault change is',
    ),
  },
  tabs: {
    open: explain(
      requirePermissionsAsync(READ),
      'the tab shows the user a file and answers only its id; the plugin of the tab kind reads the file under its own permissions',
    ),
    navigate: explain(
      requirePermissionsAsync(READ),
      'the tab shows the user a file and answers only its id; the plugin of the tab kind reads the file under its own permissions',
    ),
    close: explain(
      requirePermissions(WRITE),
      'it ends any tab of the window by id, and with it the unsaved state of that tab',
    ),
  },
  vault: {
    notes: {
      listRelatedNotes: requirePermissionsAsync(READ),
      getHeadings: requirePermissionsAsync(READ),
      findNotes: requirePermissionsAsync(READ),
      createUntitledNote: requirePermissionsAsync(WRITE),
      read: requirePermissionsAsync(READ),
      write: requirePermissionsAsync(WRITE),
      readProperties: requirePermissionsAsync(READ),
    },
    relations: {
      roles: requireNothing('the names of the roles, which hold no vault data'),
      getConnection: requirePermissionsAsync(READ),
      saveConnection: requirePermissionsAsync(WRITE),
      deleteConnectionSide: requirePermissionsAsync(WRITE),
      deleteConnection: requirePermissionsAsync(WRITE),
      join: requirePermissionsAsync(WRITE),
      createRelated: requirePermissionsAsync(WRITE),
      listBacklinks: requirePermissionsAsync(READ),
      resolveLinks: requirePermissionsAsync(READ),
    },
    files: {
      list: requirePermissionsAsync(READ),
      move: requirePermissionsAsync(WRITE),
      createFolder: requirePermissionsAsync(WRITE),
      createNote: requirePermissionsAsync(WRITE),
    },
    texts: {
      readLiveText: requirePermissionsAsync(READ),
      createEdit: requirePermissionsResult(WRITE),
    },
    buffers: {
      track: requirePermissions(WRITE),
      addFlushHandler: requirePermissions(WRITE),
    },
    vaults: {
      getState: requirePermissions(READ),
      watch: requirePermissions(READ),
      switchTo: requirePermissionsAsync(MANAGE),
    },
  },
  ui: {
    workspace: {
      window: {
        isReady: requireNothing('whether the window has finished loading'),
        openingPath: requirePermissions(READ),
        readOpeningPath: requirePermissionsAsync(READ),
        getToday: requireNothing('the current day, which holds no vault data'),
      },
      dragged: {
        paths: requirePermissions(READ),
        setPaths: requirePermissions(READ),
      },
      files: {
        open: requirePermissions(READ),
        openAt: requirePermissions(READ),
        syncPath: requirePermissions(READ),
      },
      messages: {
        showError: requireNothing('it shows the user a text of the plugin'),
        getWriter: requireNothing('it shows the user a text of the plugin'),
      },
      icons: {
        getCommand: requireNothing('an icon holds no vault data'),
        getEntry: requireNothing('an icon is drawn from the entry the plugin passes'),
      },
      getNoteInFocus: requirePermissions(READ),
    },
    window: {
      getState: requireNothing('the maximized and fullscreen flags of the window'),
      watch: requireNothing('the maximized and fullscreen flags of the window'),
      minimize: requireNothing('it moves the frame of the window and touches no vault data'),
      toggleMaximize: requireNothing('it moves the frame of the window and touches no vault data'),
      close: requireNothing('it closes the frame of the window and touches no vault data'),
    },
    commands: {
      getIds: requireNothing('the ids of the commands, which hold no vault data'),
      watch: requireNothing('the ids of the commands, which hold no vault data'),
      listItems: requirePermissionsAsync(READ),
      run: readOnlyFor(
        explain(
          requirePermissionsAsync(WRITE),
          'a command runs only if its owner is the plugin, or a slot lists it, which shows the user a row for it, or it is in the short list of application commands plugins run: limitCommands checks it; vault:read is enough for app:file.reveal and app:settings.open, which move a view',
        ),
        ([command]) => READ_ONLY_COMMANDS.has(command),
      ),
    },
    slots: {
      getEntries: requireNothing('the entries the manifests of the plugins declare'),
      watchEntries: requireNothing('the entries the manifests of the plugins declare'),
    },
    notices: {
      show: requireNothing('a notice is shown under the id of the plugin'),
      setTasks: requireNothing('the tasks are shown under the id of the plugin'),
    },
    picker: {
      open: requireNothing('it draws the list the plugin gives and holds no vault data'),
      promptText: requireNothing('it draws the question the plugin gives and holds no vault data'),
      selectOption: requireNothing('it draws the options the plugin gives and holds no vault data'),
    },
    dialogs: {
      open: requireNothing('it draws the view the plugin gives and holds no vault data'),
    },
    drops: {
      canDrop: requireNothing('it tells whether some plugin takes the types, nothing more'),
      drop: requirePermissionsAsync(WRITE),
    },
  },
  editor: { get: EDITOR_LIST, onChange: EDITOR_LIST, fail: EDITOR_LIST },
  work: {
    connections: {
      call: requireNothing(CONNECTION_GRANT),
      listRows: requireNothing(CONNECTION_GRANT),
      listRowOptions: requireNothing(CONNECTION_GRANT),
      onChange: requireNothing('it says only that the rows may have changed'),
    },
    tasks: {
      watch: requireNothing('the tasks the plugin itself reported'),
      watchAll: requirePermissions(READ),
    },
    backgroundTasks: {
      watch: requireNothing('the work of the window, which holds no vault content'),
      cancel: requirePermissionsAsync(WRITE),
      retry: requirePermissions(WRITE),
      dismiss: requireNothing('it takes a task out of the list the user sees'),
      runAction: requireNothing('it runs the fix the core checked when the item was reported'),
    },
    processes: {
      watch: requirePermissions(PLUGINS_READ),
    },
    logs: {
      getLevel: requirePermissionsAsync(LOGS_READ),
      setLevel: requirePermissionsAsync(LOGS_READ),
      watch: requirePermissions(LOGS_READ),
      write: requireNothing('the entry is filed under the id of the plugin'),
    },
  },
  health: {
    report: requireNothing('the item is filed under the id of the plugin'),
    clear: requireNothing('it clears an item of the plugin itself'),
  },
}
