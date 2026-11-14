/** A note is asked for its widened box and its headings only while it is attended to or open, whatever the frames of a move do. */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import PlexView from './PlexView.vue'
import { arrangePlex, DEFAULT_OPTIONS, interpolatePlex } from '../../lib/arrange'
import { build } from '../../fixtures/build'
import type { PlacedNode } from '../../lib/node'

const FRAMES = 8

const from = arrangePlex(build('Hub', { child: 3 }))
const to = arrangePlex(build('Hub', { child: 2, jump: 1 }))

afterEach(() => {
  document.body.replaceChildren()
})

const mountView = () => {
  const widen = vi.fn((_node: PlacedNode) => null)
  const placedHeadings = vi.fn((_node: PlacedNode) => null)
  const view = mount(PlexView, {
    attachTo: document.body,
    props: {
      namesOpacity: 0.3,
      focusedNamesOpacity: 0.9,
      frame: from,
      viewport: { width: 1200, height: 800 },
      nodeSize: DEFAULT_OPTIONS.nodeSize,
      widen,
      placedHeadings,
    },
  })
  return { view, widen, placedHeadings }
}

const play = async (view: ReturnType<typeof mountView>['view']) => {
  for (let step = 1; step <= FRAMES; step++) {
    await view.setProps({ frame: interpolatePlex(from, to, step / FRAMES) })
  }
}

describe('the sizes requested during a move', () => {
  it('are requested of no note while none is hovered', async () => {
    const { view, widen, placedHeadings } = mountView()

    await play(view)

    expect(widen).not.toHaveBeenCalled()
    expect(placedHeadings).not.toHaveBeenCalled()
  })

  it('are requested of the hovered note alone, at most once for each frame', async () => {
    const { view, widen, placedHeadings } = mountView()
    const attended = from.nodes.find((node) => node.relation === 'child')!
    await view.get(`[aria-label^="${attended.title}"]`).trigger('pointerenter')
    await view.vm.$nextTick()
    widen.mockClear()
    placedHeadings.mockClear()

    await play(view)

    expect(new Set(placedHeadings.mock.calls.map(([node]) => node.id))).toStrictEqual(new Set([attended.id]))
    expect(new Set(widen.mock.calls.map(([node]) => node.id))).toStrictEqual(new Set([attended.id]))
    expect(placedHeadings.mock.calls.length).toBeLessThanOrEqual(FRAMES)
  })
})
