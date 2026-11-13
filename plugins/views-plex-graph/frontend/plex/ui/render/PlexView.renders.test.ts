/** What a move of the pointer redraws: the notes it concerns, and nothing else. */
import { mount } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import PlexNodeView from './PlexNodeView.vue'
import PlexView from './PlexView.vue'
import { arrangePlex, DEFAULT_OPTIONS } from '../../lib/arrange'
import type { Drop } from '../../lib/arrange'
import type { PlacedNode, Position } from '../../lib/node'
import type { PlexRelation } from '../../../types'
import { build } from '../../fixtures/build'

const frame = arrangePlex(build('Hub', { parent: 2, child: 6, jump: 2 }))

const VIEWPORT = { width: 1200, height: 800 }

const createCountedView = (props: Record<string, unknown> = {}) => {
  const redrawn = new Map<string, number>()
  const view = mount(PlexView, {
    attachTo: document.body,
    props: {
      namesOpacity: 0.3,
      focusedNamesOpacity: 0.9,
      frame,
      viewport: VIEWPORT,
      nodeSize: DEFAULT_OPTIONS.nodeSize,
      highlightLinks: true,
      ...props,
    },
    global: {
      mixins: [
        {
          updated(this: { $props: { node?: PlacedNode }; $: { type: unknown } }) {
            if (this.$.type !== PlexNodeView) return
            const id = this.$props.node!.id
            redrawn.set(id, (redrawn.get(id) ?? 0) + 1)
          },
        },
      ],
    },
  })
  const reset = () => redrawn.clear()
  return { view, redrawn, reset }
}

const note = (view: ReturnType<typeof createCountedView>['view'], id: string) =>
  view.get(`[data-testid="${id}"], [aria-label^="${id}"]`)

describe('the pointer moving over a note', () => {
  it('redraws only that note, and not the others', async () => {
    const { view, redrawn, reset } = createCountedView()
    reset()

    await note(view, frame.nodes[3]!.title).trigger('pointerenter')
    await nextTick()

    expect(frame.nodes.length).toBeGreaterThan(8)
    expect([...redrawn.keys()]).toStrictEqual([frame.nodes[3]!.id])
  })

  it('does not settle the width or the headings of a note again', async () => {
    const widen = vi.fn((node: PlacedNode) => ({ width: node.width + 40, offset: 0 }))
    const hung = vi.fn(() => null)
    const { view } = createCountedView({ widen, hung })
    const widened = widen.mock.calls.length
    const hanged = hung.mock.calls.length
    expect(hanged).toBeGreaterThan(0)

    await note(view, frame.nodes[4]!.title).trigger('pointerenter')
    await nextTick()
    await note(view, frame.nodes[4]!.title).trigger('pointerleave')
    await nextTick()

    expect(widened).toBe(frame.nodes.length)
    expect(widen.mock.calls.length).toBe(widened)
    expect(hung.mock.calls.length).toBe(hanged)
  })

  it('lifts the lines of the note without redrawing the others', async () => {
    const { view, redrawn, reset } = createCountedView()
    reset()

    await note(view, frame.nodes[0]!.title).trigger('pointerenter')
    await nextTick()

    expect(view.findAll('.plex__raised .plex__edge').length).toBeGreaterThan(0)
    expect([...redrawn.keys()]).toStrictEqual([frame.nodes[0]!.id])
  })
})

describe('a frame that changes', () => {
  it('redraws the note that moved, and not the others', async () => {
    const { view, redrawn, reset } = createCountedView()
    reset()
    const moved = frame.nodes[3]!

    await view.setProps({
      frame: { ...frame, nodes: frame.nodes.map((node) => (node === moved ? { ...node, x: node.x + 10 } : node)) },
    })
    await nextTick()

    expect([...redrawn.keys()]).toStrictEqual([moved.id])
  })
})

describe('the pointer moving during a gesture or a drag', () => {
  it('redraws no note when only the end of the thread moves', async () => {
    const source = frame.nodes[0]!
    const create = { kind: 'create', from: source.id, relation: 'child' } as const
    const gesture = {
      from: ref<string | null>(source.id),
      at: ref<Position | null>({ x: 10, y: 10 }),
      outcome: ref<Drop | null>(create),
    }
    const dragging = { at: ref<Position | null>(null), relation: ref<PlexRelation | null>(null) }
    const { redrawn, reset } = createCountedView({ gesture, dragging })
    reset()

    gesture.at.value = { x: 30, y: 60 }
    gesture.outcome.value = { ...create }
    await nextTick()
    dragging.relation.value = 'child'
    dragging.at.value = { x: 5, y: 5 }
    await nextTick()
    dragging.at.value = { x: 9, y: 9 }
    await nextTick()

    expect([...redrawn.keys()]).toStrictEqual([])
  })

  it('draws the thread where the pointer is', async () => {
    const source = frame.nodes[0]!
    const at = ref<Position | null>({ x: 10, y: 10 })
    const gesture = { from: ref<string | null>(source.id), at, outcome: ref<Drop | null>(null) }
    const { view } = createCountedView({ gesture })
    const before = view.get('.plex__reach .plex__thread').attributes('d')

    at.value = { x: 200, y: 150 }
    await nextTick()

    expect(view.get('.plex__reach .plex__thread').attributes('d')).not.toBe(before)
  })
})
