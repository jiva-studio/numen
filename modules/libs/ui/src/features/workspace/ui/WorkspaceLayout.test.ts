/**
 * What the workspace does with a close, and what it hands its tabs.
 *
 * A workspace of one pane is drawn without a splitter, which is what these
 * mount: the arrangement is the model's affair and is tested there.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import WorkspaceLayout from './WorkspaceLayout.vue'
import { WorkspacePane } from './pane'
import { type Tab, type Workspace as State } from '../lib/node'
import { oneStack, sideBySide, stack, createWorkspace } from '../fixtures/build'

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
  it('is told to the caller before anything is applied', async () => {
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

  it('is applied where the caller lets it go', async () => {
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
  it('is told to the caller with the tab and the point', async () => {
    const target = mountWorkspace()

    await target
      .find('[data-workspace-tab="chat"]')
      .trigger('contextmenu', { clientX: 5, clientY: 6 })

    expect(target.emitted('tab-context-menu')).toStrictEqual([['chat', { x: 5, y: 6 }]])
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
 * A workspace is one pane or a branch of them, and a slot reaches a tab down
 * whichever of the two it was drawn as.
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
  it('emits tear-off when released outside any drop target', async () => {
    const target = mountWorkspace()
    const tab = target.find('[data-workspace-tab="plex"]')
    tab.element.dispatchEvent(
      new PointerEvent('pointerdown', { button: 0, clientX: 200, clientY: 200, bubbles: true }),
    )
    window.dispatchEvent(
      new PointerEvent('pointermove', { clientX: 10, clientY: 10, bubbles: true }),
    )
    window.dispatchEvent(new PointerEvent('pointerup', { clientX: 10, clientY: 10, bubbles: true }))

    expect(target.emitted('tear-off')).toStrictEqual([['plex', { x: 10, y: 10 }]])
  })
})
