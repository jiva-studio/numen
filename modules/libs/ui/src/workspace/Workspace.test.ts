/**
 * What the workspace does with a close, and what it hands its tabs.
 *
 * A workspace of one pane is drawn without a splitter, which is what these
 * mount: the arrangement is the model's affair and is tested there.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Workspace from './Workspace.vue'
import WorkspacePane from './render/WorkspacePane.vue'
import { type Tab, type Workspace as State } from './model'
import { oneStack, split, stack, workspaceOf } from './fixtures/build'

const TABS: readonly Tab[] = [
  { id: 'plex', title: 'Plex' },
  { id: 'chat', title: 'Chat' },
]

const mountWorkspace = (
  state: State = oneStack(),
  props: Record<string, unknown> = {},
  slots: Record<string, string> = {},
) =>
  mount(Workspace, {
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

  it('is not offered a close', () => {
    expect(closeOf(mountWorkspace(alone()), 'plex').exists()).toBe(false)
  })

  it('is not closed by a close asked for from anywhere else', async () => {
    const held = mountWorkspace(alone())
    held.findComponent(WorkspacePane).vm.$emit('close', 'plex')
    await held.vm.$nextTick()

    expect(held.emitted('close')).toBeUndefined()
    expect(held.emitted('update:modelValue')).toBeUndefined()
  })

  it('is offered a close again as soon as it is not the last', () => {
    expect(closeOf(mountWorkspace(oneStack()), 'plex').exists()).toBe(true)
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

describe('a new tab asked for', () => {
  it('carries the pane it was asked in, and opens nothing', async () => {
    const held = mountWorkspace(oneStack(), { newTab: 'New tab' })
    await held.get('[data-workspace-new]').trigger('click')

    expect(held.emitted('open')).toStrictEqual([['main']])
    expect(held.emitted('update:modelValue')).toBeUndefined()
  })

  it('names the pane under the pointer where the workspace is divided', async () => {
    const divided = workspaceOf(
      split('root', [stack('main', 'plex'), stack('aside', 'chat')]),
      'horizontal',
      'main',
    )
    const held = mountWorkspace(divided, { newTab: 'New tab' })
    const asides = held.findAllComponents(WorkspacePane)[1]
    await asides?.get('[data-workspace-new]').trigger('click')

    expect(held.emitted('open')).toStrictEqual([['aside']])
  })

  it('is offered by no strip where no word for it was given', () => {
    expect(mountWorkspace().find('[data-workspace-new]').exists()).toBe(false)
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
