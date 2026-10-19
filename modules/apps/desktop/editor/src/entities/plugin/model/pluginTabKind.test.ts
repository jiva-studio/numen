import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, type PropType } from 'vue'
import { mount } from '@vue/test-utils'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import { createPluginHost } from './host'
import {
  createDeclaredPane,
  createDeclaredTabKind,
  isDeclaredTabKind,
  pickDeclaredHooks,
  type DeclaredTabKindRegistration,
} from './pluginTabKind'
import type { PluginHostDeps } from './types'

interface Sample {
  readonly at: string
  readonly id: string
  title: string
}

const Pane = defineComponent({
  name: 'HeldPane',
  props: {
    state: { type: Object as PropType<Sample>, required: true },
  },
  setup: (props) => () => h('p', { class: 'sample' }, props.state.at),
})

const declare = (over: Partial<TabKindRegistration<Sample>> = {}): DeclaredTabKindRegistration =>
  ({
    kind: 'sample',
    component: Pane,
    title: 'Sample',
    open: (at: string, id: string): Sample => ({ at, id, title: '' }),
    ...over,
  }) as DeclaredTabKindRegistration

describe('a registration that says what its tabs hold', () => {
  it('is told apart from one that holds a note', () => {
    expect(isDeclaredTabKind(declare())).toBe(true)
    expect(isDeclaredTabKind({ kind: 'note', component: Pane })).toBe(false)
  })

  it('opens its tabs itself, under the identity the window gives them', () => {
    const kind = createDeclaredTabKind('md.numen.sample', declare())

    expect(kind.open('Graph.md', 'sample:1')).toEqual({ at: 'Graph.md', id: 'sample:1', title: '' })
    expect(kind.kind).toBe('sample')
    expect(kind.pluginId).toBe('md.numen.sample')
  })

  it('is called what it calls a tab, and its title where it calls it nothing', () => {
    const getTitle = (state: Sample) => state.title
    const named = createDeclaredTabKind('p', declare({ getTitle }))
    const plain = createDeclaredTabKind('p', declare())
    const { title: _title, ...untitled } = declare()
    const bare = createDeclaredTabKind('p', untitled)

    expect(named.getTitle?.({ at: '', id: '', title: 'Focus' })).toBe('Focus')
    expect(plain.getTitle?.({ at: '', id: '', title: 'Focus' })).toBe('Sample')
    expect(bare.getTitle?.({})).toBe('sample')
  })

  it('hands the window every hook it declares, and no other', () => {
    const navigate = vi.fn()
    const placement = { pane: 'main', order: 0 } as const
    const kind = createDeclaredTabKind('p', declare({ navigate, isTool: true, placement }))

    expect(kind.navigate).toBe(navigate)
    expect(kind.isTool).toBe(true)
    expect(kind.placement).toBe(placement)
    expect(kind).not.toHaveProperty('identity')
    expect(kind).not.toHaveProperty('onClose')
    expect(kind).not.toHaveProperty('getTarget')
  })
})

describe('the hooks a registration hands over', () => {
  it('are the ones it declares', () => {
    const onClose = () => true
    const icon = { name: 'Icon' }

    expect(pickDeclaredHooks({ kind: 'k', component: null, onClose, icon })).toStrictEqual({
      onClose,
      icon,
    })
    expect(pickDeclaredHooks({ kind: 'k', component: null })).toStrictEqual({})
  })
})

describe('the pane of a declared kind', () => {
  it('draws the component on the state of the tab, outside a shadow root and with nothing else', () => {
    const view = mount(createDeclaredPane('md.numen.sample', Pane), {
      props: { state: { at: 'Graph.md', id: 'sample:1', title: '' } },
    })

    expect(view.find('.sample').text()).toBe('Graph.md')
    expect(view.element.shadowRoot).toBeNull()
    expect(view.find('.sample').attributes()).toStrictEqual({ class: 'sample' })
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

    const sub = context.registerTabKind<Sample>({
      kind: 'sample',
      component: Pane,
      open: (at, id) => ({ at, id, title: 'T' }),
      isTool: true,
    })

    const [kind] = registered as ReturnType<typeof createDeclaredTabKind>[]
    expect(kind?.isTool).toBe(true)
    expect(kind?.open('A.md', 'sample:1')).toEqual({ at: 'A.md', id: 'sample:1', title: 'T' })
    sub.dispose()
    expect(registered).toHaveLength(0)
  })
})
