/**
 * What each service a plugin context offers needs the plugin to hold, in one table. A guarded
 * service is built from the table, not from the service: a member the table does not list is not
 * on the guarded service, and the table's type lists every member of the SDK types, so a member
 * added to the SDK fails the typecheck until it states its permission or why it needs none.
 */
import type {
  Caught,
  PluginEvents,
  PluginIdentity,
  PluginRegistrations,
  PluginSettings,
  PluginTabs,
  PluginUi,
  PluginVault,
  PluginWork,
} from '@numen/plugin-sdk'
import type { RequirePermission } from './permission'

/** What a member of a service holds: the guard passes members on without knowing their type. */
type Value = object | string | number | bigint | boolean | symbol | null | undefined | void

const READ = 'vault:read'
const WRITE = 'vault:write'
const MANAGE = 'vault:manage'
const NETWORK = 'network:outbound'

/**
 * How a refusal of a rule reaches the caller: a sync member throws, an async member rejects, and a
 * rule that lists no permission cannot refuse, so it fits either.
 */
type Mode = 'sync' | 'async' | 'none'

/** What a member needs: every permission listed, and whether a refusal throws or rejects. */
class Rule<M extends Mode = Mode> {
  constructor(
    readonly permissions: readonly string[],
    readonly mode: M,
    readonly reason: string,
  ) {}

  get isAsync(): boolean {
    return this.mode === 'async'
  }
}

function requirePermissions(...permissions: string[]): Rule<'sync'> {
  return new Rule(permissions, 'sync', '')
}

function requirePermissionsAsync(...permissions: string[]): Rule<'async'> {
  return new Rule(permissions, 'async', '')
}

function requireNothing(reason: string): Rule<'none'> {
  return new Rule([], 'none', reason)
}

function explain<M extends Mode>(rule: Rule<M>, reason: string): Rule<M> {
  return new Rule(rule.permissions, rule.mode, reason)
}

type Leaf =
  | ((...args: never[]) => Value)
  | readonly Value[]
  | { readonly value: Value }
  | string
  | number
  | boolean

/**
 * The rule a member can have, from its type: a member that returns a promise (or may) takes an
 * async rule, so a refusal rejects; any other member takes a sync rule, so a refusal throws.
 */
type RuleOf<M> = M extends (...args: never[]) => infer R
  ? [Extract<R, PromiseLike<Value>>] extends [never]
    ? Rule<'sync' | 'none'>
    : Rule<'async' | 'none'>
  : Rule<'sync' | 'none'>

/** The rules of a service: one for each member, nested for a member that is a service itself. */
export type RulesOf<S> = {
  readonly [K in keyof S]-?: NonNullable<S[K]> extends Leaf
    ? RuleOf<NonNullable<S[K]>>
    : RulesOf<NonNullable<S[K]>>
}

/** The parts of a plugin context that offer services. */
export interface GuardedParts {
  readonly plugin: Pick<PluginIdentity, 'openExternal'>
  readonly register: PluginRegistrations
  readonly settings: PluginSettings
  readonly events: PluginEvents
  readonly tabs: PluginTabs
  readonly vault: PluginVault
  readonly ui: Omit<PluginUi, 'mountView'>
  readonly work: PluginWork
}

export const SERVICE_RULES: RulesOf<GuardedParts> = {
  plugin: {
    openExternal: requirePermissionsAsync(NETWORK),
  },
  register: {
    reader: requireNothing('a reader names the extensions the manifest of the plugin declares'),
    settingsView: requireNothing('the view draws the settings of the plugin itself'),
    slotView: requireNothing('the slots are the ones the manifest of the plugin places it in'),
    tabKind: requirePermissions(READ),
    command: requireNothing('a command runs where the user or a plugin invokes it'),
    agentProvider: requirePermissions(READ),
    view: requireNothing(
      'the manifest must list the kind and the plugin hold what the kind hands its views: checked in viewPermissions',
    ),
    dropHandler: requireNothing('a handler takes dropped data and reads nothing of the vault'),
  },
  settings: {
    get: requireNothing('it reads the settings of the plugin itself'),
    onChange: requireNothing('it listens to the settings of the plugin itself'),
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
      getNeighborhood: requirePermissionsAsync(READ),
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
      removeSide: requirePermissionsAsync(WRITE),
      removeConnection: requirePermissionsAsync(WRITE),
      join: requirePermissionsAsync(WRITE),
      createRelated: requirePermissionsAsync(WRITE),
      listBacklinks: requirePermissionsAsync(READ),
    },
    files: {
      list: requirePermissionsAsync(READ),
      move: requirePermissionsAsync(WRITE),
      createFolder: requirePermissionsAsync(WRITE),
      createNote: requirePermissionsAsync(WRITE),
    },
    texts: {
      readLiveText: requirePermissionsAsync(READ),
      createEdit: requirePermissionsAsync(WRITE),
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
      ask: requirePermissions(WRITE),
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
      run: explain(
        requirePermissionsAsync(WRITE),
        'a command runs only if its owner is the plugin, or a slot lists it, which shows the user a row for it, or it is in the short list of application commands plugins run: limitCommands checks it',
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
    drops: {
      canDrop: requireNothing('it tells whether some plugin takes the types, nothing more'),
      drop: requirePermissionsAsync(WRITE),
    },
  },
  work: {
    agent: {
      ask: requirePermissions(WRITE),
      finish: explain(
        requirePermissionsAsync(WRITE),
        'it ends a conversation the agent holds, as ask starts one',
      ),
      resolveLinks: requirePermissionsAsync(READ),
      getUnreachable: requireNothing('why no agent can be reached, which holds no vault data'),
      getSetting: requirePermissions(READ),
      writeSetting: requirePermissionsAsync(WRITE),
      getModelsAt: requirePermissions(READ),
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
      openSettings: requireNothing('it opens the settings of a plugin, which holds no vault data'),
    },
  },
}

type Members = Record<string, Value>

/** What a refusal needs: the check of the permissions, and the path of the service for the message. */
export interface Guard {
  readonly check: RequirePermission
  readonly path?: string
}

function requireAll(
  rule: Rule,
  operation: string,
  check: RequirePermission,
): { readonly error: Caught } | undefined {
  try {
    for (const permission of rule.permissions) check(permission, operation)
  } catch (error) {
    return { error }
  }
  return undefined
}

function guardMember(
  service: Members,
  entry: { readonly key: string; readonly rule: Rule },
  guard: { readonly check: RequirePermission; readonly operation: string },
): PropertyDescriptor {
  const { key, rule } = entry
  const { check, operation } = guard
  if (typeof service[key] === 'function') {
    return {
      enumerable: true,
      value: (...args: never[]) => {
        const failure = requireAll(rule, operation, check)
        if (failure === undefined) return Reflect.apply(service[key] as () => Value, service, args)
        if (rule.isAsync) return Promise.reject(failure.error)
        throw failure.error
      },
    }
  }
  return {
    enumerable: true,
    get: () => {
      const failure = requireAll(rule, operation, check)
      if (failure !== undefined) throw failure.error
      return service[key]
    },
  }
}

/**
 * The service as the plugin may use it: only the members the rules list, each checked when it is
 * used. A member the service has and the rules do not list is not reachable; a member the service
 * lacks is absent. `guard.path` prefixes the operation named in a refusal.
 */
export function guardService<S extends object>(service: S, rules: RulesOf<S>, guard: Guard): S {
  const source = service as Members
  const guarded: Members = {}
  for (const [key, rule] of Object.entries<Rule | object>(rules)) {
    const member = source[key]
    if (member === undefined) continue
    const operation = guard.path === undefined ? key : `${guard.path}.${key}`
    if (rule instanceof Rule) {
      const descriptor = guardMember(source, { key, rule }, { check: guard.check, operation })
      Object.defineProperty(guarded, key, descriptor)
    } else if (typeof member === 'object' && member !== null) {
      guarded[key] = guardService(member, rule, {
        check: guard.check,
        path: operation,
      })
    }
  }
  return guarded as S
}
