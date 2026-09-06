/**
 * What the workspace does with a close, and what it hands its tabs.
 *
 * A workspace of one pane is drawn without a splitter, which is what these
 * mount: the arrangement is the model's affair and is tested there.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import WorkspaceLayout from './WorkspaceLayout.vue'
import WorkspacePane from './render/WorkspacePane.vue'
import { type Tab, type Workspace as State } from './node'
import { oneStack, sideBySide, stack, workspaceOf } from './fixtures/build'

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

const closeOf = (held: ReturnType<typeof mountWorkspace>, tab: string) =>
  held.find(`[data-workspace-tab="${tab}"] button`)

describe('a close', () => {
  it('is told to the caller before anything is applied', async () => {
    let told: unknown
    const held = mountWorkspace(oneStack(), {
      onClose: () => {
        told = held.emitted('update:modelValue')
      },
    })

    await closeOf(held, 'chat').trigger('click')

    expect(told).toBeUndefined()
    expect(held.emitted('close')?.[0]?.[0]).toBe('chat')
  })

  it('is applied where the caller lets it go', async () => {
    const held = mountWorkspace()
    await closeOf(held, 'chat').trigger('click')

    const after = held.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.tabs).toStrictEqual(['plex'])
  })

  it('leaves the tab standing where the caller holds it', async () => {
    const hold = vi.fn()
    const held = mountWorkspace(oneStack(), {
      onClose: (_tab: string, take: () => void) => {
        take()
        hold()
      },
    })

    await closeOf(held, 'chat').trigger('click')

    expect(hold).toHaveBeenCalledOnce()
    expect(held.emitted('update:modelValue')).toBeUndefined()
    expect(held.findAll('[data-workspace-tab]')).toHaveLength(2)
  })
})

describe('the last tab in the workspace', () => {
  const alone = () => workspaceOf(stack('main', 'plex'))

  it('is offered a close, as every other tab is', () => {
    expect(closeOf(mountWorkspace(alone()), 'plex').exists()).toBe(true)
  })

  it('leaves one pane holding nothing where it is closed', async () => {
    const held = mountWorkspace(alone())
    await closeOf(held, 'plex').trigger('click')

    const after = held.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.tabs).toStrictEqual([])
    expect(held.findAll('[data-workspace-tab]')).toHaveLength(0)
    expect(held.findComponent(WorkspacePane).exists()).toBe(true)
  })

  // The silence stands on the arrangement the caller wrote back, not on
  // anything the workspace kept to itself: a caller holding the tab open sees
  // none of this.
  it('leaves the silence in its place', async () => {
    const held = mountWorkspace(alone(), {}, { silence: '<p class="quiet">Nothing here</p>' })
    await closeOf(held, 'plex').trigger('click')

    const after = held.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.tabs).toStrictEqual([])
    await held.setProps({ modelValue: after })

    expect(held.find('.quiet').exists()).toBe(true)
    expect(held.find('[data-workspace-strip]').exists()).toBe(false)
  })
})

describe('what a tab carries', () => {
  it('reaches the strip from the tabs the caller declared', () => {
    const held = mountWorkspace(oneStack(), {
      tabs: [
        { id: 'plex', title: 'Plex' },
        { id: 'chat', title: 'Chat', mark: 'unsaved' },
      ],
    })

    expect(held.find('[data-workspace-tab="chat"] .tab__mark').attributes('aria-label')).toBe(
      'unsaved',
    )
    expect(held.find('[data-workspace-tab="plex"] .tab__mark').exists()).toBe(false)
  })

  it('is drawn by the caller where the caller draws one', () => {
    const held = mountWorkspace(
      oneStack(),
      {
        tabs: [
          { id: 'plex', title: 'Plex' },
          { id: 'chat', title: 'Chat', mark: 'stuck' },
        ],
      },
      { mark: '<i class="mine">{{ params.id }}: {{ params.mark }}</i>' },
    )

    expect(held.find('.mine').text()).toBe('chat: stuck')
    expect(held.find('.tab__mark').exists()).toBe(false)
  })
})

/**
 * A workspace is one pane or a branch of them, and a slot reaches a tab down
 * whichever of the two it was drawn as.
 */
describe('what is drawn before a tab’s name', () => {
  const drawn = (state: State) =>
    mountWorkspace(state, {}, { icon: '<i class="mine">{{ params.id }}</i>' })

  it('reaches the tabs of a workspace of one pane', () => {
    const held = drawn(oneStack())

    expect(held.findAll('.mine').map((one) => one.text())).toStrictEqual(['plex', 'chat'])
  })

  it('reaches the tabs of every pane of a workspace that is split', () => {
    const held = drawn(sideBySide())

    expect(held.findAll('.mine').map((one) => one.text())).toStrictEqual(['plex', 'chat'])
  })

  it('is nothing where the caller draws none, and takes no room', () => {
    const held = mountWorkspace(sideBySide())

    expect(held.find('.tab__icon').exists()).toBe(false)
  })
})

describe('the strip under a keyboard', () => {
  it('shows what an arrow reaches', async () => {
    const held = mountWorkspace()
    await held.find('[data-workspace-tab="plex"]').trigger('keydown', { key: 'ArrowRight' })

    const after = held.emitted('update:modelValue')?.[0]?.[0] as State
    expect(after.root.kind === 'pane' && after.root.active).toBe('chat')
    expect(held.emitted('activate')?.[0]?.[0]).toBe('chat')
  })
})
