/**
 * What the workspace does with a close, and what it hands its tabs.
 *
 * A workspace of one pane is drawn without a splitter, which is what these
 * mount: the arrangement is the model's affair and is tested there.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, onMounted, onUnmounted, watchEffect } from 'vue'
import WorkspaceLayout from './WorkspaceLayout.vue'
import { WorkspacePane } from './pane'
import { type Tab, type Workspace as State } from '../lib/node'
import { createTabHosts } from '../model/hosts'
import { oneStack, sideBySide, stack, createWorkspace } from '../fixtures/build'
import { dropTab } from '../lib/edit'

const TABS: readonly Tab[] = [
  { id: 'plex', title: 'Plex' },
  { id: 'chat', title: 'Chat' },
]

const mountWorkspace = (
  state: State = oneStack(),
  props: Record<string, unknown> = {},
  slots: Record<string, string> = {},
) =>
  mount(WorkspaceLayout, {
    attachTo: document.body,
    props: { modelValue: state, tabs: TABS, ...props },
    slots,
  })

const findCloseButton = (wrapper: ReturnType<typeof mountWorkspace>, tab: string) =>
  wrapper.find(`[data-workspace-tab="${tab}"] button`)

describe('a close', () => {
  it('is reported to the caller before anything is applied', async () => {
    let notifiedEvent: unknown
    const target = mountWorkspace(oneStack(), {
      onClose: () => {
        notifiedEvent = target.emitted('update:modelValue')
      },
    })

    await findCloseButton(target, 'chat').trigger('click')

    expect(notifiedEvent).toBeUndefined()
    expect(target.emitted('close')?.[0]?.[0]).toBe('chat')
  })

  it('is applied where the caller releases it', async () => {
    const target = mountWorkspace()
    await findCloseButton(target, 'chat').trigger('click')

    const after = target.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.tabs).toStrictEqual(['plex'])
  })

  it('leaves the tab standing where the caller holds it', async () => {
    const hold = vi.fn()
    const target = mountWorkspace(oneStack(), {
      onClose: (_tab: string, take: () => void) => {
        take()
        hold()
      },
    })

    await findCloseButton(target, 'chat').trigger('click')

    expect(hold).toHaveBeenCalledOnce()
    expect(target.emitted('update:modelValue')).toBeUndefined()
    expect(target.findAll('[data-workspace-tab]')).toHaveLength(2)
  })
})

describe('the last tab in the workspace', () => {
  const createLoneTab = () => createWorkspace(stack('main', 'plex'))

  it('is offered a close, as every other tab is', () => {
    expect(findCloseButton(mountWorkspace(createLoneTab()), 'plex').exists()).toBe(true)
  })

  it('leaves one pane holding nothing where it is closed', async () => {
    const tab = mountWorkspace(createLoneTab())
    await findCloseButton(tab, 'plex').trigger('click')

    const after = tab.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.tabs).toStrictEqual([])
    expect(tab.findAll('[data-workspace-tab]')).toHaveLength(0)
    expect(tab.findComponent(WorkspacePane).exists()).toBe(true)
  })

  // The silence relies on the arrangement the caller wrote back, not on
  // anything the workspace kept to itself: a caller holding the tab open sees
  // none of this.
  it('leaves the silence in its place', async () => {
    const target = mountWorkspace(
      createLoneTab(),
      {},
      { silence: '<p class="quiet">Nothing here</p>' },
    )
    await findCloseButton(target, 'plex').trigger('click')

    const after = target.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.tabs).toStrictEqual([])
    await target.setProps({ modelValue: after })

    expect(target.find('.quiet').exists()).toBe(true)
    expect(target.find('[data-workspace-strip]').exists()).toBe(false)
  })
})

describe('a menu requested for on a tab', () => {
  it('is reported to the caller with the tab and the point', async () => {
    const target = mountWorkspace()

    await target
      .find('[data-workspace-tab="chat"]')
      .trigger('contextmenu', { clientX: 5, clientY: 6 })

    expect(target.emitted('tab-context-menu')).toStrictEqual([['chat', { x: 5, y: 6 }]])
  })
})

describe('the menu of a tab', () => {
  const MENU = {
    tab: 'chat',
    at: { x: 5, y: 6 },
    items: [
      { id: 'detach', text: 'Detach tab' },
      { id: 'org.example.atlas/command/graph.showNote', text: 'Show note in plex', group: '1' },
    ],
  }

  const getRows = () => [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')]

  it('is drawn with the rows the caller gives, and nowhere while it gives none', async () => {
    const target = mountWorkspace(oneStack(), { tabMenu: MENU })
    await nextTick()

    expect(getRows().map((one) => one.textContent?.trim())).toStrictEqual([
      'Detach tab',
      'Show note in plex',
    ])

    await target.setProps({ tabMenu: null })
    expect(getRows()).toStrictEqual([])
    target.unmount()
  })

  it('emits the id of the chosen row with the id of its tab', async () => {
    const target = mountWorkspace(oneStack(), { tabMenu: MENU })
    await nextTick()

    getRows()[1]?.click()

    expect(target.emitted('tab-menu-choose')).toStrictEqual([
      ['chat', 'org.example.atlas/command/graph.showNote'],
    ])
    target.unmount()
  })

  it('emits the dismissal of the menu and the id of the highlighted row', async () => {
    const target = mountWorkspace(oneStack(), { tabMenu: MENU })
    await nextTick()
    const menu = target.findComponent({ name: 'Menu' })

    menu.vm.$emit('dismiss')
    menu.vm.$emit('highlight', 'detach')

    expect(target.emitted('tab-menu-dismiss')).toHaveLength(1)
    expect(target.emitted('tab-menu-highlight')).toStrictEqual([['detach']])
    target.unmount()
  })

  it('draws the icon the caller draws for a row', async () => {
    const target = mountWorkspace(
      oneStack(),
      { tabMenu: MENU },
      {
        'tab-menu-icon':
          '<template #tab-menu-icon="{ id }"><i class="mark">{{ id }}</i></template>',
      },
    )
    await nextTick()

    expect(getRows()[0]?.querySelector('.mark')?.textContent).toBe('detach')
    target.unmount()
  })
})

describe('what a tab carries', () => {
  it('reaches the strip from the tabs the caller declared', () => {
    const target = mountWorkspace(oneStack(), {
      tabs: [
        { id: 'plex', title: 'Plex' },
        { id: 'chat', title: 'Chat', mark: 'unsaved' },
      ],
    })

    expect(target.find('[data-workspace-tab="chat"] .tab__mark').attributes('aria-label')).toBe(
      'unsaved',
    )
    expect(target.find('[data-workspace-tab="plex"] .tab__mark').exists()).toBe(false)
  })

  it('is drawn by the caller where the caller draws one', () => {
    const target = mountWorkspace(
      oneStack(),
      {
        tabs: [
          { id: 'plex', title: 'Plex' },
          { id: 'chat', title: 'Chat', mark: 'stuck' },
        ],
      },
      { mark: '<i class="mine">{{ params.id }}: {{ params.mark }}</i>' },
    )

    expect(target.find('.mine').text()).toBe('chat: stuck')
    expect(target.find('.tab__mark').exists()).toBe(false)
  })
})

/**
 * A workspace is one pane or a branch of panes, and a slot resolves to a tab
 * whichever of the two it was rendered as.
 */
describe('what is drawn before a tab’s name', () => {
  const mountWithIcon = (state: State) =>
    mountWorkspace(state, {}, { icon: '<i class="mine">{{ params.id }}</i>' })

  it('reaches the tabs of a workspace of one pane', () => {
    const target = mountWithIcon(oneStack())

    expect(target.findAll('.mine').map((one) => one.text())).toStrictEqual(['plex', 'chat'])
  })

  it('reaches the tabs of every pane of a workspace that is split', () => {
    const target = mountWithIcon(sideBySide())

    expect(target.findAll('.mine').map((one) => one.text())).toStrictEqual(['plex', 'chat'])
  })

  it('is nothing where the caller draws none, and takes no room', () => {
    const target = mountWorkspace(sideBySide())

    expect(target.find('.tab__icon').exists()).toBe(false)
  })
})

describe('the strip under a keyboard', () => {
  it('shows what an arrow reaches', async () => {
    const target = mountWorkspace()
    await target.find('[data-workspace-tab="plex"]').trigger('keydown', { key: 'ArrowRight' })

    const after = target.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.active).toBe('chat')
    expect(target.emitted('activate')?.[0]?.[0]).toBe('chat')
  })
})

describe('a tab dragged out of the workspace', () => {
  beforeEach(() => {
    document.elementFromPoint = () => null
  })

  afterEach(() => {
    Reflect.deleteProperty(document, 'elementFromPoint')
  })

  it('emits tear-off on pointerup outside the viewport', async () => {
    const target = mountWorkspace()
    const tab = target.find('[data-workspace-tab="plex"]')
    tab.element.dispatchEvent(
      new PointerEvent('pointerdown', { button: 0, clientX: 200, clientY: 200, bubbles: true }),
    )
    window.dispatchEvent(
      new PointerEvent('pointermove', { clientX: 10, clientY: 10, bubbles: true }),
    )
    window.dispatchEvent(
      new PointerEvent('pointerup', { clientX: -10, clientY: 10, bubbles: true }),
    )

    expect(target.emitted('tear-off')).toStrictEqual([['plex', { x: -10, y: 10 }]])
  })
})

describe('a tab moved to another pane', () => {
  it('keeps the content it was rendered with', async () => {
    const lifecycle = { mounted: 0, unmounted: 0 }
    const Content = defineComponent({
      setup() {
        onMounted(() => lifecycle.mounted++)
        onUnmounted(() => lifecycle.unmounted++)
        return () => h('p', { 'data-content': 'plex' }, 'plex')
      },
    })
    const target = mount(WorkspaceLayout, {
      attachTo: document.body,
      props: { modelValue: sideBySide(), tabs: TABS },
      slots: { tab: ({ id }: { id: string }) => (id === 'plex' ? h(Content) : h('div')) },
    })
    await nextTick()
    const element = target.element.querySelector('[data-content="plex"]')

    const moved = dropTab(
      target.props('modelValue') as State,
      { tab: 'plex', onto: 'aside', side: 'center' },
      () => 'made',
    )
    await target.setProps({ modelValue: moved })
    await nextTick()
    await nextTick()

    expect(target.element.querySelector('[data-content="plex"]')).toBe(element)
    expect(target.element.querySelector('[data-workspace-pane="aside"]')?.contains(element)).toBe(
      true,
    )
    expect(lifecycle).toStrictEqual({ mounted: 1, unmounted: 0 })
    target.unmount()
  })

  it('tracks tab host elements across attach and detach', async () => {
    const hosts = createTabHosts()
    const first = document.createElement('div')
    const second = document.createElement('div')
    let reads = 0
    const stop = watchEffect(() => {
      hosts.getElement('plex')
      reads += 1
    })
    expect(reads).toBe(1)

    hosts.attach('plex', 'main', first)
    await nextTick()
    expect(hosts.getElement('plex')).toBe(first)
    expect(reads).toBe(2)

    hosts.attach('plex', 'main', first)
    await nextTick()
    expect(reads).toBe(2)

    hosts.attach('plex', 'main', second)
    await nextTick()
    expect(hosts.getElement('plex')).toBe(second)
    expect(reads).toBe(3)

    hosts.detach('plex', 'aside')
    expect(hosts.getElement('plex')).toBe(second)
    hosts.detach('unknown', 'main')

    hosts.detach('plex', 'main')
    expect(hosts.getElement('plex')).toBeUndefined()
    stop()
  })
})
