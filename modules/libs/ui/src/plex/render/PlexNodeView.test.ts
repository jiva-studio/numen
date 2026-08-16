/**
 * What a node decides on its own: whether it can be chosen, what it tells a
 * screen reader, and what the handle does that the box must not.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexNodeView from './PlexNodeView.vue'
import type { PlacedNode, PlexSeat } from '../model'

const nodeAt = (over: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  title: 'A node',
  seat: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
  ...over,
})

const mountNode = (over: Partial<PlacedNode> = {}) =>
  mount(PlexNodeView, { props: { node: nodeAt(over) } })

/** A pointer arrives, which is the only thing that offers a handle. */
const hover = async (node: ReturnType<typeof mountNode>) => {
  await node.trigger('pointerenter')
  return node
}

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
    const node = mountNode({ seat: 'focus' })
    await node.trigger('click')
    await node.trigger('keydown', { key: 'Enter' })
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is not a tab stop, but is still announced', () => {
    const node = mountNode({ seat: 'focus' })
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
    expect(node.attributes('aria-label')).toBe('A node, child')
  })
})

describe('the handle', () => {
  it('is not there until a pointer is, and goes when it leaves', async () => {
    const node = mountNode()
    expect(node.find('.plex__handle').exists()).toBe(false)

    await hover(node)
    expect(node.find('.plex__handle').exists()).toBe(true)

    await node.trigger('pointerleave')
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('is not offered where reaching out is not on offer', async () => {
    const node = mount(PlexNodeView, { props: { node: nodeAt(), standing: 'closed' } })
    await node.trigger('pointerenter')
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('is not offered by a node that is on its way in or out', async () => {
    const node = await hover(mountNode({ opacity: 0.4 }))
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('stays put once a gesture has left from it, hand or no hand', () => {
    // The pointer is somewhere else entirely by then, dragging the thread.
    const node = mount(PlexNodeView, { props: { node: nodeAt(), standing: 'source' } })
    expect(node.find('.plex__handle').exists()).toBe(true)
  })

  it('starts a gesture rather than choosing the node it sits on', async () => {
    const node = await hover(mountNode())
    await node.get('.plex__handle').trigger('pointerdown')
    expect(node.emitted('reach')).toHaveLength(1)
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('sits on the trailing edge, halfway down', async () => {
    const at = (await hover(mountNode({ width: 144 }))).get('.plex__handle-at')
    expect(at.attributes('transform')).toBe('translate(72 0)')
  })
})

describe('the icon', () => {
  it('is whatever the slot was handed, and sits beside the title', () => {
    const node = mount(PlexNodeView, {
      props: { node: nodeAt() },
      slots: { icon: '<i class="glyph" />' },
    })
    const title = node.get('.plex__title')
    expect(title.find('.plex__icon .glyph').exists()).toBe(true)
    expect(title.get('.plex__title-text').text()).toBe('A node')
  })

  it('takes up no room at all when the slot is left empty', () => {
    expect(mountNode().find('.plex__icon').exists()).toBe(false)
  })
})

describe('a node that is not there yet', () => {
  const mountGhost = () =>
    mount(PlexNodeView, {
      props: { node: nodeAt({ title: 'parent', seat: 'parent' }), standing: 'ghost' },
    })

  it('cannot be chosen, and is not a tab stop', async () => {
    const ghost = mountGhost()
    await ghost.trigger('click')
    await ghost.trigger('keydown', { key: 'Enter' })
    expect(ghost.emitted('activate')).toBeUndefined()
    expect(ghost.attributes('tabindex')).toBe('-1')
  })

  it('is told to nobody: it has no name, and is not something to press', () => {
    const ghost = mountGhost()
    expect(ghost.attributes('aria-hidden')).toBe('true')
    expect(ghost.attributes('aria-label')).toBeUndefined()
    expect(ghost.attributes('role')).toBeUndefined()
  })

  it('offers no handle of its own, however long a hand rests on it', async () => {
    const ghost = await hover(mountGhost())
    expect(ghost.find('.plex__handle').exists()).toBe(false)
  })

  it('says the seat it would take, because it has nothing else to say', () => {
    expect(mountGhost().get('.plex__title-text').text()).toBe('parent')
  })
})

describe('what a node is drawn as', () => {
  it('carries its own seat, as a class and as a hue', () => {
    for (const seat of ['focus', 'parent', 'child', 'jump', 'sibling'] as PlexSeat[]) {
      const node = mountNode({ seat })
      expect(node.classes()).toContain(`plex__node--${seat}`)
      expect(node.attributes('style')).toContain(`var(--numen-seat-${seat})`)
    }
  })

  it('carries what it is to the gesture as a class of its own', () => {
    expect(mountNode().classes()).toContain('plex__node--open')
    const target = mount(PlexNodeView, { props: { node: nodeAt(), standing: 'target' } })
    expect(target.classes()).toContain('plex__node--target')
  })

  it('is a box the size the arrangement asked for, in pixels', () => {
    const rect = mountNode({ width: 200, height: 50 }).get('rect')
    expect(Number(rect.attributes('width'))).toBe(200)
    expect(Number(rect.attributes('height'))).toBe(50)
  })

  it('has a name for a screen reader even with no title at all', () => {
    expect(mountNode({ title: '' }).attributes('aria-label')).toBe('Untitled, child')
  })
})
