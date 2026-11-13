import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, type PropType } from 'vue'
import { mount } from '@vue/test-utils'
import type { PluginUi, PluginVault, TabKindRegistration } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import {
  clearPluginStyles,
  loadPluginStyle,
  registerPluginStyle,
  unregisterPluginStyle,
} from '../styles'
import {
  createDeclaredTabKind,
  isDeclaredTabKind,
  pickDeclaredParts,
  type DeclaredTabKindRegistration,
} from './pluginTabKind'
import type { PluginHostDeps } from './types'
import { STUB_COMPONENT } from '../testing/stubComponent'

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
  setup: (props) => () => h('p', { class: 'member' }, props.state.at),
})

const declare = (overrides: Partial<TabKindRegistration<Held>> = {}): DeclaredTabKindRegistration =>
  ({
    kind: 'member',
    component: Pane,
    title: 'Held',
    open: (folderPath: string, id: string): Held => ({ at: folderPath, id, title: '' }),
    ...overrides,
  }) as DeclaredTabKindRegistration

const createDeclaredPane = (pluginId: string, component: unknown) =>
  createDeclaredTabKind(pluginId, declare({ component: component as never })).pane

describe('a registration that says what its tabs hold', () => {
  it('is distinguished from one that holds a note', () => {
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
    expect(kind.kind).toBe('member')
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
    expect(bareKind.getTitle?.({})).toBe('member')
  })

  it('passes the window every hook it declares, and no other', () => {
    const navigate = vi.fn()
    const receiveText = vi.fn()
    const placement = { pane: 'main', order: 0 } as const
    const kind = createDeclaredTabKind(
      'p',
      declare({ navigate, receiveText, isTool: true, placement }),
    )

    expect(kind.address?.navigate).toBe(navigate)
    expect(kind.events?.receiveText).toBe(receiveText)
    expect(kind.display?.isTool).toBe(true)
    expect(kind.display?.placement).toBe(placement)
    expect(kind.address).not.toHaveProperty('identity')
    expect(kind.events).not.toHaveProperty('onClose')
    expect(kind.address).not.toHaveProperty('getTarget')
  })
})

describe('the hooks a registration passes over', () => {
  it('are the ones it declares', () => {
    const onClose = () => true
    const icon = { name: 'Icon' }

    expect(
      pickDeclaredParts({ kind: 'k', component: STUB_COMPONENT, onClose, icon }),
    ).toStrictEqual({ display: { icon }, events: { onClose }, address: {} })
    expect(pickDeclaredParts({ kind: 'k', component: STUB_COMPONENT })).toStrictEqual({
      display: {},
      events: {},
      address: {},
    })
  })

  it('include what the window asks of the kind that edits a file', () => {
    const hooks = {
      getMark: () => 'unsaved',
      isAsking: () => true,
      flushPending: vi.fn(() => Promise.resolve()),
      createFile: async () => ({ ok: true as const, value: { path: 'a.md' } }),
    }

    const parts = pickDeclaredParts({ kind: 'k', component: STUB_COMPONENT, ...hooks })

    expect({ ...parts.display, ...parts.events, ...parts.address }).toStrictEqual(hooks)
  })
})

describe('the pane of a declared kind', () => {
  it('draws the component on the state of the tab, outside a shadow root and with nothing else', () => {
    const view = mount(createDeclaredPane('md.numen.sample', Pane), {
      props: { state: { at: 'Graph.md', id: 'target: 1', title: '' } },
    })

    expect(view.find('.member').text()).toBe('Graph.md')
    expect(view.find('link').exists()).toBe(false)
    expect(view.element.shadowRoot).toBeNull()
    expect(view.find('.member').attributes()).toStrictEqual({ class: 'member' })
    view.unmount()
  })

  it('adopts a constructed stylesheet on mount and releases it on unmount without rendering a link', async () => {
    class FakeSheet {
      cssText = ''
      replaceSync(text: string) {
        this.cssText = text
      }
    }
    vi.stubGlobal('CSSStyleSheet', FakeSheet)
    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
    try {
      await loadPluginStyle(
        'md.numen.sample',
        '/plugin/md.numen.sample/dist/styles.css',
        async () => ({ ok: true, text: async () => '.member {}' }) as unknown as Response,
      )
      const view = mount(createDeclaredPane('md.numen.sample', Pane), {
        props: { state: { at: 'Graph.md', id: 'target: 1', title: '' } },
      })
      expect(view.find('link').exists()).toBe(false)
      expect(document.adoptedStyleSheets.length).toBe(1)

      view.unmount()
      expect(document.adoptedStyleSheets.length).toBe(0)
    } finally {
      clearPluginStyles()
      vi.unstubAllGlobals()
      Reflect.deleteProperty(document, 'adoptedStyleSheets')
    }
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
    expect(view.find('.member').text()).toBe('Graph.md')
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
      target: 'desktop.editor',
      registerKind: (kind) => {
        registered.push(kind)
        return () => registered.splice(registered.indexOf(kind), 1)
      },
      registerExtensionReader: vi.fn(() => vi.fn()),
      openTab: vi.fn(() => Promise.resolve('tab')),
    }
    const context = createPluginHost(deps).lifecycle.createContext(
      'md.numen.sample',
      createMockManifest('md.numen.sample', {
        name: 'Sample',
        permissions: ['vault:read'],
        tabKinds: ['member'],
      }),
    )

    const sub = context.register.tabKind<Held>({
      kind: 'member',
      component: Pane,
      open: (folderPath, id) => ({ at: folderPath, id, title: 'T' }),
      isTool: true,
    })

    const [kind] = registered as ReturnType<typeof createDeclaredTabKind>[]
    expect(kind?.display?.isTool).toBe(true)
    expect(kind?.open('A.md', 'target: 1')).toEqual({ at: 'A.md', id: 'target: 1', title: 'T' })
    sub.dispose()
    expect(registered).toHaveLength(0)
  })
})

describe('the services a plugin context is given', () => {
  const createDeps = (overrides: Partial<PluginHostDeps> = {}): PluginHostDeps => ({
    target: 'desktop.editor',
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve('tab'),
    ...overrides,
  })

  it('are the window and the notes the host has made by the time the plugin activates', async () => {
    const focus = { path: 'Orchard/Pruning.md', title: 'Pruning' }
    const workspace = { getNoteInFocus: () => focus } as unknown as NonNullable<
      PluginUi['workspace']
    >
    const read = vi.fn(async () => ({ body: 'Pruning notes.' }))
    const notes = { read } as unknown as NonNullable<PluginVault['notes']>
    const host = createPluginHost(
      createDeps({ getWorkspace: () => workspace, getNotes: () => notes }),
    )

    const context = host.lifecycle.createContext(
      'md.numen.sample',
      createMockManifest('md.numen.sample', { name: 'Sample', permissions: ['vault:read'] }),
    )

    expect(context.ui.workspace?.getNoteInFocus()).toStrictEqual(focus)
    await expect(context.vault.notes!.read('Orchard/Pruning.md')).resolves.toStrictEqual({
      body: 'Pruning notes.',
    })
    expect(read).toHaveBeenCalledWith('Orchard/Pruning.md')
  })

  it('are absent where the host offers none', () => {
    const context = createPluginHost(createDeps()).lifecycle.createContext('md.numen.sample')

    expect(context).not.toHaveProperty('workspace')
    expect(context).not.toHaveProperty('notes')
  })

  it('tell every plugin listening for a change to the vault, and a failing one is reported', async () => {
    const onError = vi.fn()
    const host = createPluginHost(createDeps({ onError }))
    const received: unknown[] = []
    const readText = createMockManifest('md.numen.one', {
      name: 'One',
      permissions: ['vault:read'],
    })
    host.lifecycle.createContext('md.numen.one', readText).events.onVaultChange((event) => {
      received.push(event)
    })
    host.lifecycle.createContext('md.numen.two', readText).events.onVaultChange(() => {
      throw new Error('no')
    })

    host.notifyVaultChange({ paths: ['A.md'] })
    await new Promise((isDone) => setTimeout(isDone, 0))

    expect(received).toStrictEqual([{ paths: ['A.md'] }])
    expect(onError).toHaveBeenCalledWith('md.numen.two', expect.any(Error))
  })
})
