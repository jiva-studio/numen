/**
 * The strips at the top of the window: which of them move the window, while no title bar is drawn
 * above the workspace.
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
      expect(getStrip(wrapper, pane).hasAttribute('data-window-drag')).toBe(true)
    }
  })

  it('does not move the window by a strip below another pane', () => {
    const column = createWorkspace(
      split('column', [stack('upper', 'plex'), stack('lower', 'rain')]),
      'vertical',
    )
    const wrapper = mountWorkspace(column, EDGE)

    expect(getStrip(wrapper, 'upper').hasAttribute('data-window-drag')).toBe(true)
    expect(getStrip(wrapper, 'lower').hasAttribute('data-window-drag')).toBe(false)
  })
})

describe('a workspace under a title bar', () => {
  it('moves the window by no strip', () => {
    const wrapper = mountWorkspace(sideBySide())

    for (const pane of ['main', 'aside']) {
      expect(getStrip(wrapper, pane).hasAttribute('data-window-drag')).toBe(false)
    }
  })
})
