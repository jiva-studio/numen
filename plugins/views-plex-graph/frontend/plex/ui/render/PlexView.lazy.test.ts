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
  const hung = vi.fn((_node: PlacedNode) => null)
  const view = mount(PlexView, {
    attachTo: document.body,
    props: {
      namesOpacity: 0.3,
      focusedNamesOpacity: 0.9,
      frame: from,
      viewport: { width: 1200, height: 800 },
      nodeSize: DEFAULT_OPTIONS.nodeSize,
      widen,
      hung,
    },
  })
  return { view, widen, hung }
}

const play = async (view: ReturnType<typeof mountView>['view']) => {
  for (let step = 1; step <= FRAMES; step++) {
    await view.setProps({ frame: interpolatePlex(from, to, step / FRAMES) })
  }
}

describe('the boxes and headings asked for during a move', () => {
  it('are asked of no note while none is attended to', async () => {
    const { view, widen, hung } = mountView()

    await play(view)

    expect(widen).not.toHaveBeenCalled()
    expect(hung).not.toHaveBeenCalled()
  })

  it('are asked of the attended note alone, at most once for each frame', async () => {
    const { view, widen, hung } = mountView()
    const attended = from.nodes.find((node) => node.relation === 'child')!
    await view.get(`[aria-label^="${attended.title}"]`).trigger('pointerenter')
    await view.vm.$nextTick()
    widen.mockClear()
    hung.mockClear()

    await play(view)

    expect(new Set(hung.mock.calls.map(([node]) => node.id))).toStrictEqual(new Set([attended.id]))
    expect(new Set(widen.mock.calls.map(([node]) => node.id))).toStrictEqual(new Set([attended.id]))
    expect(hung.mock.calls.length).toBeLessThanOrEqual(FRAMES)
  })
})
