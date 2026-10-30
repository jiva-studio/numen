import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, type PropType } from 'vue'
import { mount } from '@vue/test-utils'
import type { PluginContext, TabKindRegistration } from '@numen/plugin-sdk'
import { createPluginHost } from './host'
import { registerPluginStyle, unregisterPluginStyle } from '@numen/plugin-host'
import {
  createDeclaredPane,
  createDeclaredTabKind,
  isDeclaredTabKind,
  pickDeclaredHooks,
  type DeclaredTabKindRegistration,
} from './pluginTabKind'
import type { PluginHostDeps } from './types'

interface Held {
  readonly at: string
  readonly id: string
  title: string
}

const Pane = defineComponent({
  name: 'HeldPane',
  props: {
    state: { type: Object as PropType<Held>, required: true },
  },
  setup: (props) => () => h('p', { class: 'held' }, props.state.at),
})

const declare = (overrides: Partial<TabKindRegistration<Held>> = {}): DeclaredTabKindRegistration =>
  ({
    kind: 'held',
    component: Pane,
    title: 'Held',
    open: (folderPath: string, id: string): Held => ({ at: folderPath, id, title: '' }),
    ...overrides,
  }) as DeclaredTabKindRegistration

describe('a registration that says what its tabs hold', () => {
  it('is told apart from one that holds a note', () => {
    expect(isDeclaredTabKind(declare())).toBe(true)
    expect(isDeclaredTabKind({ kind: 'note', component: Pane })).toBe(false)
  })

  it('opens its tabs itself, under the identity the window gives them', () => {
    const kind = createDeclaredTabKind('md.numen.sample', declare())

    expect(kind.open('Graph.md', 'target: 1')).toEqual({
      at: 'Graph.md',
      id: 'target: 1',
      title: '',
    })
    expect(kind.kind).toBe('held')
    expect(kind.pluginId).toBe('md.numen.sample')
  })

  it('is called what it calls a tab, and its title where it calls it nothing', () => {
    const getTitle = (state: Held) => state.title
    const named = createDeclaredTabKind('p', declare({ getTitle }))
    const plainKind = createDeclaredTabKind('p', declare())
    const { title: _title, ...untitled } = declare()
    const bareKind = createDeclaredTabKind('p', untitled)

    expect(named.getTitle?.({ at: '', id: '', title: 'Focus' })).toBe('Focus')
    expect(plainKind.getTitle?.({ at: '', id: '', title: 'Focus' })).toBe('Held')
    expect(bareKind.getTitle?.({})).toBe('held')
  })

  it('passes the window every hook it declares, and no other', () => {
    const navigate = vi.fn()
    const receiveText = vi.fn()
    const placement = { pane: 'main', order: 0 } as const
    const kind = createDeclaredTabKind(
      'p',
      declare({ navigate, receiveText, isTool: true, placement }),
    )

    expect(kind.navigate).toBe(navigate)
    expect(kind.receiveText).toBe(receiveText)
    expect(kind.isTool).toBe(true)
    expect(kind.placement).toBe(placement)
    expect(kind).not.toHaveProperty('identity')
    expect(kind).not.toHaveProperty('onClose')
    expect(kind).not.toHaveProperty('getTarget')
  })
})

describe('the hooks a registration passes over', () => {
  it('are the ones it declares', () => {
    const onClose = () => true
    const icon = { name: 'Icon' }

    expect(pickDeclaredHooks({ kind: 'k', component: null, onClose, icon })).toStrictEqual({
      onClose,
      icon,
    })
    expect(pickDeclaredHooks({ kind: 'k', component: null })).toStrictEqual({})
  })

  it('include what the window asks of the kind that edits a file', () => {
    const hooks = {
      getMark: () => 'unsaved',
      isAsking: () => true,
      flushPending: vi.fn(() => Promise.resolve()),
      createFile: async () => ({ ok: true as const, value: { path: 'a.md' } }),
    }

    expect(pickDeclaredHooks({ kind: 'k', component: null, ...hooks })).toStrictEqual(hooks)
  })
})

describe('the pane of a declared kind', () => {
  it('draws the component on the state of the tab, outside a shadow root and with nothing else', () => {
    const view = mount(createDeclaredPane('md.numen.sample', Pane), {
      props: { state: { at: 'Graph.md', id: 'target: 1', title: '' } },
    })

    expect(view.find('.held').text()).toBe('Graph.md')
    expect(view.element.shadowRoot).toBeNull()
    expect(view.find('.held').attributes()).toStrictEqual({ class: 'held' })
  })

  it('renders the plugin stylesheet next to the component', () => {
    registerPluginStyle('md.numen.sample', '/plugin/md.numen.sample/dist/styles.css')
    const view = mount(createDeclaredPane('md.numen.sample', Pane), {
      props: { state: { at: 'Graph.md', id: 'target: 1', title: '' } },
    })
    unregisterPluginStyle('md.numen.sample')

    const link = view.find('link[rel="stylesheet"]')
    expect(link.attributes('href')).toBe('/plugin/md.numen.sample/dist/styles.css')
    expect(link.attributes('data-plugin-id')).toBe('md.numen.sample')
    expect(view.find('.held').text()).toBe('Graph.md')
    expect(document.head.querySelector('link[data-plugin-id="md.numen.sample"]')).toBeNull()
  })

  it('says the plugin failed where the component throws', async () => {
    const Broken = defineComponent({
      setup: () => () => {
        throw new Error('the graph is gone')
      },
    })
    const view = mount(createDeclaredPane('md.numen.sample', Broken), { props: { state: {} } })
    await view.vm.$nextTick()

    expect(view.text()).toContain('Plugin "md.numen.sample" crashed')
    expect(view.text()).toContain('the graph is gone')
  })

  it('draws nothing where the plugin gives no component', () => {
    const view = mount(createDeclaredPane('p', null), { props: { state: {} } })

    expect(view.html()).toBe('')
  })
})

describe('a declared kind registered through a plugin context', () => {
  it('reaches the window as the kind the plugin declared', () => {
    const registered: unknown[] = []
    const deps: PluginHostDeps = {
      registerKind: (kind) => {
        registered.push(kind)
        return () => registered.splice(registered.indexOf(kind), 1)
      },
      registerExtensionReader: vi.fn(() => vi.fn()),
      openTab: vi.fn(() => Promise.resolve('tab')),
    }
    const context = createPluginHost(deps).createContext('md.numen.sample')

    const sub = context.registerTabKind<Held>({
      kind: 'held',
      component: Pane,
      open: (folderPath, id) => ({ at: folderPath, id, title: 'T' }),
      isTool: true,
    })

    const [kind] = registered as ReturnType<typeof createDeclaredTabKind>[]
    expect(kind?.isTool).toBe(true)
    expect(kind?.open('A.md', 'target: 1')).toEqual({ at: 'A.md', id: 'target: 1', title: 'T' })
    sub.dispose()
    expect(registered).toHaveLength(0)
  })
})

describe('the services a plugin context is given', () => {
  const createDeps = (overrides: Partial<PluginHostDeps> = {}): PluginHostDeps => ({
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve('tab'),
    ...overrides,
  })

  it('are the window and the notes the host has made by the time the plugin activates', () => {
    const workspace = { name: 'workspace' } as unknown as NonNullable<PluginContext['workspace']>
    const notes = { name: 'notes' } as unknown as NonNullable<PluginContext['notes']>
    const host = createPluginHost(
      createDeps({ getWorkspace: () => workspace, getNotes: () => notes }),
    )

    const context = host.createContext('md.numen.sample')

    expect(context.workspace).toMatchObject(workspace)
    expect(context.notes).toMatchObject(notes)
  })

  it('are absent where the host offers none', () => {
    const context = createPluginHost(createDeps()).createContext('md.numen.sample')

    expect(context).not.toHaveProperty('workspace')
    expect(context).not.toHaveProperty('notes')
  })

  it('tell every plugin listening for a change to the vault, and a failing one is reported', async () => {
    const onError = vi.fn()
    const host = createPluginHost(createDeps({ onError }))
    const received: unknown[] = []
    const readText = {
      id: 'md.numen.one',
      name: 'One',
      version: '1.0.0',
      permissions: ['vault:read'],
    }
    host.createContext('md.numen.one', readText).onVaultChange((event) => {
      received.push(event)
    })
    host.createContext('md.numen.two', readText).onVaultChange(() => {
      throw new Error('no')
    })

    host.notifyVaultChange({ paths: ['A.md'] })
    await new Promise((isDone) => setTimeout(isDone, 0))

    expect(received).toStrictEqual([{ paths: ['A.md'] }])
    expect(onError).toHaveBeenCalledWith('md.numen.two', expect.any(Error))
  })
})
