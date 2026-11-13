/**
 * What the services of a plugin context refuse a plugin that lacks their permission, and that no
 * path reaches a service member the table of permissions does not list.
 */
import { describe, expect, expectTypeOf, it, vi } from 'vitest'
import {
  type BackgroundTaskHost,
  type BufferHost,
  type CommandsHost,
  type DropHost,
  type NoticeHost,
  type PluginContext,
  type PluginEvents,
  type PluginRegistrations,
  type PluginSettings,
  type PluginTabs,
  type PluginUi,
  type PluginVault,
  type PluginWork,
  type SlotHost,
  type TaskHost,
  type VaultFiles,
  type VaultNotes,
  type VaultRelations,
  type VaultTexts,
  type VaultsHost,
  type WindowHost,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { createPluginContext } from '../context'
import { disposeSubscriptions } from './subscriptions'
import { SERVICE_RULES, type RulesOf } from './servicePermissions'
import type { PluginContextHost } from '../types'
import { createTestContextHost, type ContextHostParts } from '../../testing/contextHost'

interface Rule {
  readonly permissions: readonly string[]
  readonly isAsync: boolean
  readonly reason: string
}

type Tree = { [key: string]: Rule | Tree }

const isRule = (value: Rule | Tree): value is Rule => Array.isArray((value as Rule).permissions)

function createFakeServices(rules: Tree): Record<string, unknown> {
  const fake: Record<string, unknown> = {}
  for (const [key, rule] of Object.entries(rules)) {
    fake[key] = isRule(rule) ? vi.fn(() => Promise.resolve([])) : createFakeServices(rule)
  }
  return fake
}

interface Leaf {
  readonly path: readonly string[]
  readonly rule: Rule
}

function listLeaves(rules: Tree, path: readonly string[] = []): Leaf[] {
  return Object.entries(rules).flatMap(([key, rule]) =>
    isRule(rule) ? [{ path: [...path, key], rule }] : listLeaves(rule, [...path, key]),
  )
}

const asTree = (rules: unknown): Tree => rules as Tree

const MANIFEST = (permissions: readonly string[]) => ({
  id: 'md.numen.sample',
  name: 'Sample',
  version: '1.0.0',
  permissions,
})

function createHost(
  fakes: Record<string, Record<string, unknown>>,
  overrides: ContextHostParts = {},
): PluginContextHost {
  const vault = fakes.vault as Record<string, never>
  const ui = fakes.ui as Record<string, never>
  const work = fakes.work as Record<string, never>
  return createTestContextHost({
    ...overrides,
    vault: {
      getNotes: () => vault.notes,
      getRelations: () => vault.relations,
      getFiles: () => vault.files,
      getTexts: () => vault.texts,
      getBuffers: () => vault.buffers,
      getVaults: () => vault.vaults,
      ...overrides.vault,
    },
    views: { drops: ui.drops as never, ...overrides.views },
    window: {
      getWorkspace: () => ui.workspace,
      getWindow: () => ui.window,
      getCommands: () => ui.commands,
      isCommandListed: () => true,
      getSlots: () => ui.slots as never,
      getNotices: () => ui.notices as never,
      getTasks: () => work.tasks,
      getBackgroundTasks: () => work.backgroundTasks,
      getProcesses: () => work.processes,
      ...overrides.window,
    },
  })
}

const SERVICE_PARTS = ['vault', 'ui', 'work'] as const

function createFakes() {
  const rules = SERVICE_RULES as unknown as Record<string, Tree>
  return Object.fromEntries(SERVICE_PARTS.map((part) => [part, createFakeServices(asTree(rules[part]))]))
}

function pick(context: PluginContext, path: readonly string[]): { target: unknown; key: string } {
  let target: unknown = context
  for (const step of path.slice(0, -1)) target = (target as Record<string, unknown>)[step]
  return { target, key: path[path.length - 1] ?? '' }
}

/** The arguments of a member whose wrapper in the context reads them; every other member takes two paths. */
const ARGUMENTS: Readonly<Record<string, readonly unknown[]>> = {
  'vault.buffers.track': [{ flush: async () => undefined }],
}

function call(context: PluginContext, path: readonly string[]): unknown {
  const { target, key } = pick(context, path)
  return (target as Record<string, (...args: unknown[]) => unknown>)[key]?.(
    ...(ARGUMENTS[path.join('.')] ?? ['a.md', 'b.md']),
  )
}

const MEMBERS = SERVICE_PARTS.flatMap((part) =>
  listLeaves(asTree((SERVICE_RULES as unknown as Record<string, unknown>)[part])).map((leaf) => ({
    ...leaf,
    path: [part, ...leaf.path],
  })),
).filter((leaf) => leaf.rule.permissions.length > 0)

const label = (path: readonly string[]) => path.join('.')

describe('every member of a service that lists a permission', () => {
  it.each(MEMBERS.map((one) => [label(one.path), one] as const))(
    'refuses %s without it and reaches the service with it',
    async (_, member) => {
      const fakes = createFakes()
      const host = createHost(fakes)
      const missing = member.rule.permissions[0] ?? ''
      const denied = createPluginContext('md.numen.sample', host, MANIFEST([]))

      const failure = member.rule.isAsync
        ? await Promise.resolve()
            .then(() => call(denied, member.path))
            .catch((error: unknown) => error)
        : (() => {
            try {
              call(denied, member.path)
            } catch (error) {
              return error
            }
            return undefined
          })()
      expect(String(failure)).toContain(`lacks required permission "${missing}"`)

      const allowed = createPluginContext(
        'md.numen.sample',
        host,
        MANIFEST(member.rule.permissions),
      )
      const reached = call(allowed, member.path)
      await Promise.resolve(reached)
      const service = pick({ ...fakes, plugin: undefined } as unknown as PluginContext, member.path)
      expect((service.target as Record<string, unknown>)[service.key]).toHaveBeenCalled()
    },
  )
})

describe('the services of a plugin context', () => {
  it('list a permission for the members that reach the vault or act for the user', () => {
    const needs = (path: string) =>
      MEMBERS.find((one) => label(one.path) === path)?.rule.permissions

    expect(needs('vault.relations.resolveLinks')).toStrictEqual(['vault:read'])
    expect(needs('vault.vaults.getState')).toStrictEqual(['vault:read'])
    expect(needs('vault.vaults.switchTo')).toStrictEqual(['vault:manage'])
    expect(needs('ui.commands.run')).toStrictEqual(['vault:write'])
    expect(SERVICE_RULES.tabs.close.permissions).toStrictEqual(['vault:write'])
    expect(SERVICE_RULES.events.emitFileSelect.permissions).toStrictEqual(
      SERVICE_RULES.events.emitVaultChange.permissions,
    )
    expect(needs('ui.workspace.files.open')).toStrictEqual(['vault:read'])
    expect(needs('ui.drops.drop')).toStrictEqual(['vault:write'])
  })

  it('leave out a member of a service the table does not list', () => {
    const notes = { read: vi.fn(), purge: vi.fn() }
    const host = createHost(createFakes(), {
      vault: { getNotes: () => notes as unknown as VaultNotes },
    })
    const context = createPluginContext(
      'md.numen.sample',
      host,
      MANIFEST(['vault:read', 'vault:write']),
    )

    expect(Object.keys(context.vault.notes ?? {})).toStrictEqual(['read'])
  })

  it('leave out a member a host service answers to any name, such as a proxy', () => {
    const everything = new Proxy({}, { get: () => vi.fn() })
    const host = createHost(createFakes(), { vault: { getNotes: () => everything as VaultNotes } })
    const context = createPluginContext(
      'md.numen.sample',
      host,
      MANIFEST(['vault:read', 'vault:write']),
    )

    expect(Object.keys(context.vault.notes ?? {})).toStrictEqual(
      Object.keys(SERVICE_RULES.vault.notes),
    )
    expect((context.vault.notes as unknown as Record<string, unknown>).purge).toBeUndefined()
  })

  it('check a member a host service defines as a getter', async () => {
    const track = vi.fn()
    const buffers = { addFlushHandler: vi.fn() }
    Object.defineProperty(buffers, 'track', { get: () => track })
    const host = createHost(createFakes(), {
      vault: { getBuffers: () => buffers as unknown as BufferHost },
    })
    const context = createPluginContext('md.numen.sample', host, MANIFEST(['vault:read']))

    expect(() => context.vault.buffers?.track({} as never)).toThrowError(
      'lacks required permission "vault:write" for buffers.track',
    )
    expect(track).not.toHaveBeenCalled()
  })

  it('check a service the window offers after the context was made', () => {
    const services: { vaults?: VaultsHost } = {}
    const host = createHost(createFakes(), { vault: { getVaults: () => services.vaults } })
    const context = createPluginContext('md.numen.sample', host, MANIFEST([]))
    expect(context.vault.vaults).toBeUndefined()

    services.vaults = {
      getState: vi.fn(),
      watch: vi.fn(),
      switchTo: vi.fn(),
    } as unknown as VaultsHost
    const later = createPluginContext('md.numen.sample', host, MANIFEST([]))

    expect(() => later.vault.vaults?.getState()).toThrowError('vault:read')
  })

  it('lists the same members as the SDK types, no more and no fewer', () => {
    type Rules = typeof SERVICE_RULES
    expectTypeOf<keyof Rules['register']>().toEqualTypeOf<keyof PluginRegistrations>()
    expectTypeOf<keyof Rules['settings']>().toEqualTypeOf<keyof PluginSettings>()
    expectTypeOf<keyof Rules['events']>().toEqualTypeOf<keyof PluginEvents>()
    expectTypeOf<keyof Rules['tabs']>().toEqualTypeOf<keyof PluginTabs>()
    expectTypeOf<keyof Rules['vault']>().toEqualTypeOf<keyof PluginVault>()
    expectTypeOf<keyof Rules['vault']['notes']>().toEqualTypeOf<keyof VaultNotes>()
    expectTypeOf<keyof Rules['vault']['relations']>().toEqualTypeOf<keyof VaultRelations>()
    expectTypeOf<keyof Rules['vault']['files']>().toEqualTypeOf<keyof VaultFiles>()
    expectTypeOf<keyof Rules['vault']['texts']>().toEqualTypeOf<keyof VaultTexts>()
    expectTypeOf<keyof Rules['vault']['buffers']>().toEqualTypeOf<keyof BufferHost>()
    expectTypeOf<keyof Rules['vault']['vaults']>().toEqualTypeOf<keyof VaultsHost>()
    expectTypeOf<keyof Rules['ui']>().toEqualTypeOf<Exclude<keyof PluginUi, 'mountView'>>()
    expectTypeOf<keyof Rules['ui']['workspace']>().toEqualTypeOf<keyof WorkspaceHost>()
    expectTypeOf<keyof Rules['ui']['window']>().toEqualTypeOf<keyof WindowHost>()
    expectTypeOf<keyof Rules['ui']['commands']>().toEqualTypeOf<keyof CommandsHost>()
    expectTypeOf<keyof Rules['ui']['slots']>().toEqualTypeOf<keyof SlotHost>()
    expectTypeOf<keyof Rules['ui']['notices']>().toEqualTypeOf<keyof NoticeHost>()
    expectTypeOf<keyof Rules['ui']['drops']>().toEqualTypeOf<keyof DropHost>()
    expectTypeOf<keyof Rules['work']>().toEqualTypeOf<keyof PluginWork>()
    expectTypeOf<keyof Rules['work']['tasks']>().toEqualTypeOf<keyof TaskHost>()
    expectTypeOf<keyof Rules['work']['backgroundTasks']>().toEqualTypeOf<keyof BackgroundTaskHost>()
  })

  it('takes the async flag of a rule from the type of the member', () => {
    type SyncRule = typeof SERVICE_RULES.vault.buffers.track
    type AsyncRule = typeof SERVICE_RULES.vault.notes.read
    type Fits<Rule, Member> = [Rule] extends [RulesOf<Member>[keyof Member]] ? true : false

    expectTypeOf<Fits<SyncRule, { read(): Promise<string> }>>().toEqualTypeOf<false>()
    expectTypeOf<Fits<SyncRule, { read(): Promise<string> | string }>>().toEqualTypeOf<false>()
    expectTypeOf<Fits<AsyncRule, { peek(): string }>>().toEqualTypeOf<false>()
    expectTypeOf<Fits<AsyncRule, { read(): Promise<string> }>>().toEqualTypeOf<true>()
    expectTypeOf<Fits<SyncRule, { peek(): string }>>().toEqualTypeOf<true>()
  })

  it('rejects, and never throws, from every member whose type returns a promise', async () => {
    const asyncMembers = MEMBERS.filter((one) => one.rule.isAsync)
    expect(asyncMembers.length).toBeGreaterThan(0)
    for (const member of asyncMembers) {
      const denied = createPluginContext('md.numen.sample', createHost(createFakes()), MANIFEST([]))
      let returned: unknown
      expect(() => (returned = call(denied, member.path)), label(member.path)).not.toThrow()
      await expect(returned, label(member.path)).rejects.toThrowError('lacks required permission')
    }
  })

  it('states why a member needs no permission', () => {
    const free = SERVICE_PARTS.flatMap((part) =>
      listLeaves(asTree((SERVICE_RULES as unknown as Record<string, unknown>)[part])),
    ).filter((leaf) => leaf.rule.permissions.length === 0)

    expect(free.length).toBeGreaterThan(0)
    expect(free.filter((leaf) => leaf.rule.reason.trim() === '').map((l) => l.path)).toStrictEqual(
      [],
    )
  })

  it('check a property a host service defines, such as the opening path', () => {
    const openingPath = { value: 'Notes/Example.md' }
    const fakes = createFakes()
    Object.assign(fakes.ui?.workspace as object, {
      window: { openingPath, isReady: { value: true } },
    })
    const host = createHost(fakes)

    const denied = createPluginContext('md.numen.sample', host, MANIFEST([]))
    const allowed = createPluginContext('md.numen.sample', host, MANIFEST(['vault:read']))

    expect(() => denied.ui.workspace?.window.openingPath).toThrowError('vault:read')
    expect(denied.ui.workspace?.window.isReady.value).toBe(true)
    expect(allowed.ui.workspace?.window.openingPath).toBe(openingPath)
  })
})

describe('the subscriptions of a plugin', () => {
  const createSubscriptions = () => {
    const context = createPluginContext('md.numen.sample', createHost(createFakes()), MANIFEST([]))
    return context.plugin.subscriptions
  }

  it('cannot be spliced, popped, shifted or reassigned', () => {
    const subscriptions = createSubscriptions() as unknown as Record<string, unknown>
    const dispose = vi.fn()
    ;(subscriptions.push as (...a: unknown[]) => void)({ dispose })

    expect(subscriptions.splice).toBeUndefined()
    expect(subscriptions.pop).toBeUndefined()
    expect(subscriptions.shift).toBeUndefined()
    expect(() => {
      subscriptions.length = 0
    }).toThrowError()
    expect(() => {
      subscriptions.push = () => 0
    }).toThrowError()
    expect(subscriptions.length).toBe(1)
  })

  it('still releases what the plugin added when the host disposes them', () => {
    const subscriptions = createSubscriptions()
    const first = vi.fn()
    const second = vi.fn()
    subscriptions.push({ dispose: first }, { dispose: second })

    disposeSubscriptions(subscriptions)

    expect(first).toHaveBeenCalledOnce()
    expect(second).toHaveBeenCalledOnce()
    expect(subscriptions.length).toBe(0)
  })
})
