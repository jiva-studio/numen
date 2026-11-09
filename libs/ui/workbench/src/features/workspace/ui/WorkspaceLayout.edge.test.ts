/**
 * The strips at the top of the window: what moves the window and what keeps room for the window
 * buttons, while no title bar is drawn above the workspace.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import WorkspaceLayout from './WorkspaceLayout.vue'
import type { Tab, Workspace } from '../lib/node'
import { createWorkspace, sideBySide, split, stack } from '../fixtures/build'

const TABS: readonly Tab[] = [
  { id: 'plex', title: 'Plex' },
  { id: 'chat', title: 'Chat' },
  { id: 'rain', title: 'Rain log' },
]

const EDGE = { leftInset: 80, rightInset: 0 }

const mountWorkspace = (state: Workspace, windowEdge?: typeof EDGE) =>
  mount(WorkspaceLayout, {
    attachTo: document.body,
    props: { modelValue: state, tabs: TABS, ...(windowEdge ? { windowEdge } : {}) },
  })

const getStrip = (wrapper: ReturnType<typeof mountWorkspace>, pane: string) =>
  wrapper.find(`[data-workspace-strip="${pane}"]`).element as HTMLElement

afterEach(() => {
  document.body.innerHTML = ''
})

describe('a workspace at the top of the window', () => {
  it('moves the window by the strips at the top edge', () => {
    const wrapper = mountWorkspace(sideBySide(), EDGE)

    for (const pane of ['main', 'aside']) {
      expect(getStrip(wrapper, pane).style.getPropertyValue('--wails-draggable')).toBe('drag')
      expect(getStrip(wrapper, pane).hasAttribute('data-window-drag')).toBe(true)
    }
  })

  it('does not move the window by a tab', () => {
    const wrapper = mountWorkspace(sideBySide(), EDGE)

    const tab = wrapper.find('[data-workspace-tab="plex"]').element as HTMLElement
    expect(tab.style.getPropertyValue('--wails-draggable')).toBe('no-drag')
  })

  it('keeps the room of the window buttons in the strip at the top-left corner only', () => {
    const wrapper = mountWorkspace(sideBySide(), EDGE)

    expect(getStrip(wrapper, 'main').style.paddingInlineStart).toBe('80px')
    expect(getStrip(wrapper, 'aside').style.paddingInlineStart).toBe('0px')
  })

  it('does not move the window by a strip below another pane', () => {
    const column = createWorkspace(
      split('column', [stack('upper', 'plex'), stack('lower', 'rain')]),
      'vertical',
    )
    const wrapper = mountWorkspace(column, EDGE)

    expect(getStrip(wrapper, 'upper').hasAttribute('data-window-drag')).toBe(true)
    expect(getStrip(wrapper, 'lower').hasAttribute('data-window-drag')).toBe(false)
    expect(getStrip(wrapper, 'lower').style.getPropertyValue('--wails-draggable')).toBe('')
  })

  it('moves the room to the new top-left strip when the layout changes', async () => {
    const wrapper = mountWorkspace(sideBySide(), EDGE)

    await wrapper.setProps({
      modelValue: createWorkspace(
        split('root', [stack('aside', 'chat'), stack('main', 'plex')], [0.5, 0.5]),
      ),
    })

    expect(getStrip(wrapper, 'aside').style.paddingInlineStart).toBe('80px')
    expect(getStrip(wrapper, 'main').style.paddingInlineStart).toBe('0px')
  })
})

describe('a workspace under a title bar', () => {
  it('moves the window by no strip and keeps no room', () => {
    const wrapper = mountWorkspace(sideBySide())

    for (const pane of ['main', 'aside']) {
      expect(getStrip(wrapper, pane).hasAttribute('data-window-drag')).toBe(false)
      expect(getStrip(wrapper, pane).style.getPropertyValue('--wails-draggable')).toBe('')
      expect(getStrip(wrapper, pane).style.paddingInlineStart).toBe('')
    }
  })
})
