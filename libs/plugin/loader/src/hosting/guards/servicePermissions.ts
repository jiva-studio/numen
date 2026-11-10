/**
 * What each service a plugin context offers needs the plugin to hold, in one table. A guarded
 * service is built from the table, not from the service: a member the table does not list is not
 * on the guarded service, and the table's type lists every member of the SDK types, so a member
 * added to the SDK fails the typecheck until it states its permission or why it needs none.
 */
import type {
  PluginEvents,
  PluginRegistrations,
  PluginSettings,
  PluginTabs,
  PluginUi,
  PluginVault,
  PluginWork,
} from '@numen/plugin-sdk'
import type { RequirePermission } from './permission'

const READ = 'vault:read'
const WRITE = 'vault:write'
const MANAGE = 'vault:manage'

/** What a member needs: every permission listed, and whether a refusal throws or rejects. */
export class Rule {
  constructor(
    readonly permissions: readonly string[],
    readonly isAsync: boolean,
    readonly reason: string,
  ) {}
}

/** A member that throws where the plugin lacks a permission. */
function requirePermissions(...permissions: string[]): Rule {
  return new Rule(permissions, false, '')
}

/** A member that returns a promise, which rejects where the plugin lacks a permission. */
function requirePermissionsAsync(...permissions: string[]): Rule {
  return new Rule(permissions, true, '')
}

/** A member that needs no permission; the reason is the fact that makes it safe. */
function requireNothing(reason: string): Rule {
  return new Rule([], false, reason)
}

type Leaf =
  | ((...args: never[]) => unknown)
  | readonly unknown[]
  | { readonly value: unknown }
  | string
  | number
  | boolean

/** The rules of a service: one for each member, nested for a member that is a service itself. */
export type RulesOf<S> = {
  readonly [K in keyof S]-?: NonNullable<S[K]> extends Leaf ? Rule : RulesOf<NonNullable<S[K]>>
}

/** The parts of a plugin context that offer services. */
export interface GuardedParts {
  readonly register: PluginRegistrations
  readonly settings: PluginSettings
  readonly events: PluginEvents
  readonly tabs: PluginTabs
  readonly vault: PluginVault
  readonly ui: Omit<PluginUi, 'mountView'>
  readonly work: PluginWork
}

export const SERVICE_RULES: RulesOf<GuardedParts> = {
  register: {
    reader: requireNothing('a reader names the extensions the manifest of the plugin declares'),
    settingsView: requireNothing('the view draws the settings of the plugin itself'),
    slotView: requireNothing('the slots are the ones the manifest of the plugin places it in'),
    tabKind: requirePermissions(READ),
    command: requireNothing('a command runs where the user or a plugin invokes it'),
    agentProvider: requirePermissions(READ),
    view: requireNothing('the view registry checks the permissions of whoever mounts the view'),
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
    emitFileSelect: requirePermissions(READ),
  },
  tabs: {
    open: requirePermissions(READ),
    navigate: requirePermissions(READ),
    close: requirePermissions(READ),
  },
  vault: {
    notes: {
      getNeighbourhood: requirePermissionsAsync(READ),
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
      readLiveText: requirePermissions(READ),
      createEdit: requirePermissions(WRITE),
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
      run: requirePermissionsAsync(WRITE),
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
      finish: requirePermissions(READ),
      resolveLinks: requirePermissions(READ),
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

type Members = Record<string, unknown>

/** What a refusal needs: the check of the permissions, and the path of the service for the message. */
export interface Guard {
  readonly check: RequirePermission
  readonly path?: string
}

function requireAll(
  rule: Rule,
  operation: string,
  check: RequirePermission,
): { readonly error: unknown } | undefined {
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
      value: (...args: unknown[]) => {
        const refusal = requireAll(rule, operation, check)
        if (refusal === undefined) return Reflect.apply(service[key] as () => unknown, service, args)
        if (rule.isAsync) return Promise.reject(refusal.error)
        throw refusal.error
      },
    }
  }
  return {
    enumerable: true,
    get: () => {
      const refusal = requireAll(rule, operation, check)
      if (refusal !== undefined) throw refusal.error
      return service[key]
    },
  }
}

/**
 * The service as the plugin may use it: only the members the rules list, each checked when it is
 * used. A member the service has and the rules do not list is not reachable; a member the service
 * lacks is absent. `guard.path` prefixes the operation named in a refusal.
 */
export function guardService<S extends object>(
  service: S,
  rules: RulesOf<S>,
  guard: Guard,
): S {
  const source = service as Members
  const guarded: Members = {}
  for (const [key, rule] of Object.entries(rules as unknown as Record<string, Rule | object>)) {
    const member = source[key]
    if (member === undefined) continue
    const operation = guard.path === undefined ? key : `${guard.path}.${key}`
    if (rule instanceof Rule) {
      const descriptor = guardMember(source, { key, rule }, { check: guard.check, operation })
      Object.defineProperty(guarded, key, descriptor)
    } else if (typeof member === 'object' && member !== null) {
      guarded[key] = guardService(member, rule as RulesOf<object>, {
        check: guard.check,
        path: operation,
      })
    }
  }
  return guarded as S
}
