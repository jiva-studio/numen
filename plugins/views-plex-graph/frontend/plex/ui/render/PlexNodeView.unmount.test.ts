/** A note that goes while it is open no longer holds the rest. */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import PlexNodeView from './PlexNodeView.vue'
import { DEFAULT_BEHAVIOR } from '../../model/behavior'
import { stubClock } from '../../../fixtures/clock'
import type { PlacedNode } from '../../lib/node'

const node: PlacedNode = {
  id: 'one',
  title: 'A node',
  relation: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

afterEach(() => {
  vi.useRealTimers()
})

describe('a note that is removed', () => {
  it('says it is no longer resting', async () => {
    vi.useFakeTimers()
    const world = stubClock()
    const onSettle = vi.fn()
    const view = mount(PlexNodeView, {
      props: {
        onSettle,
        node,
        expanded: { width: 400, offset: 0 },
        behavior: { ...DEFAULT_BEHAVIOR, expansionDelay: 500, clock: world.clock },
      },
    })
    await view.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(500)
    world.run()
    await view.vm.$nextTick()
    expect(onSettle.mock.calls).toStrictEqual([[true]])

    view.unmount()

    expect(onSettle.mock.calls).toStrictEqual([[true], [false]])
  })
})
