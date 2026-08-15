/**
 * What a node decides on its own: whether it can be chosen, what it tells a
 * screen reader, and what the handle does that the box must not.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexNodeView from './PlexNodeView.vue'
import type { PlacedNode, PlexRole } from '../model'

const nodeAt = (over: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  label: 'A thought',
  role: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
  ...over,
})

const mountNode = (over: Partial<PlacedNode> = {}, offering = false) =>
  mount(PlexNodeView, { props: { node: nodeAt(over), offering } })

describe('a node that is not fully there', () => {
  const halfway = { opacity: 0.4 }

  it('cannot be chosen by clicking it', async () => {
    const node = mountNode(halfway)
    await node.trigger('click')
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('cannot be chosen from the keyboard either', async () => {
    const node = mountNode(halfway)
    await node.trigger('keydown', { key: 'Enter' })
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is not stopped at by tab, and is not announced', () => {
    const node = mountNode(halfway)
    expect(node.attributes('tabindex')).toBe('-1')
    expect(node.attributes('aria-hidden')).toBe('true')
  })
})

describe('the node in focus', () => {
  it('cannot be chosen, because it is where the reader already is', async () => {
    const node = mountNode({ role: 'focus' })
    await node.trigger('click')
    await node.trigger('keydown', { key: 'Enter' })
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is not a tab stop, but is still announced', () => {
    const node = mountNode({ role: 'focus' })
    expect(node.attributes('tabindex')).toBe('-1')
    expect(node.attributes('aria-hidden')).toBeUndefined()
    expect(node.attributes('role')).toBe('img')
  })
})

describe('a node that can be chosen', () => {
  it('is chosen by a click', async () => {
    const node = mountNode()
    await node.trigger('click')
    expect(node.emitted('activate')).toHaveLength(1)
  })

  it('is chosen by Enter and by the space bar', async () => {
    const node = mountNode()
    await node.trigger('keydown', { key: 'Enter' })
    await node.trigger('keydown', { key: ' ' })
    expect(node.emitted('activate')).toHaveLength(2)
  })

  it('is left alone by any other key', async () => {
    const node = mountNode()
    await node.trigger('keydown', { key: 'a' })
    await node.trigger('keydown', { key: 'ArrowDown' })
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is a tab stop, and is announced as something to press', () => {
    const node = mountNode()
    expect(node.attributes('tabindex')).toBe('0')
    expect(node.attributes('role')).toBe('button')
    expect(node.attributes('aria-label')).toBe('A thought, child')
  })
})

describe('the handle', () => {
  it('is drawn only when the node is told to offer one', () => {
    expect(mountNode().find('.plex__handle').exists()).toBe(false)
    expect(mountNode({}, true).find('.plex__handle').exists()).toBe(true)
  })

  it('starts a gesture rather than choosing the node it sits on', async () => {
    const node = mountNode({}, true)
    await node.get('.plex__handle').trigger('pointerdown')
    expect(node.emitted('reach')).toHaveLength(1)
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('sits on the trailing edge, halfway down', () => {
    const handle = mountNode({ width: 144 }, true).get('.plex__handle')
    expect(Number(handle.attributes('cx'))).toBe(72)
    expect(Number(handle.attributes('cy'))).toBe(0)
  })
})

describe('what a node says about a pointer', () => {
  it('reports arriving and leaving, so the plex knows where one is', async () => {
    const node = mountNode()
    await node.trigger('pointerenter')
    await node.trigger('pointerleave')
    expect(node.emitted('hover')).toStrictEqual([[true], [false]])
  })
})

describe('what a node is drawn as', () => {
  it('carries its own role, as a class and as a hue', () => {
    for (const role of ['focus', 'parent', 'child', 'jump', 'sibling'] as PlexRole[]) {
      const node = mountNode({ role })
      expect(node.classes()).toContain(`plex__node--${role}`)
      expect(node.attributes('style')).toContain(`var(--numen-role-${role})`)
    }
  })

  it('is marked when a gesture would land a link on it', () => {
    expect(mountNode().classes()).not.toContain('plex__node--aimed')
    const aimed = mount(PlexNodeView, { props: { node: nodeAt(), aimed: true } })
    expect(aimed.classes()).toContain('plex__node--aimed')
  })

  it('is a box the size the arrangement asked for, in pixels', () => {
    const rect = mountNode({ width: 200, height: 50 }).get('rect')
    expect(Number(rect.attributes('width'))).toBe(200)
    expect(Number(rect.attributes('height'))).toBe(50)
  })

  it('has a name for a screen reader even with no label at all', () => {
    expect(mountNode({ label: '' }).attributes('aria-label')).toBe('Untitled, child')
  })
})
