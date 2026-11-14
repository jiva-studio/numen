/** The handle of a box that opens keeps inside the window. */
import { effectScope } from 'vue'
import { describe, expect, it } from 'vitest'
import { useOpenBox } from './expansion'
import { HANDLE_RADIUS, type PlacedNode } from '../../lib/node'
import { stubClock } from '../../../fixtures/clock'

const node: PlacedNode = {
  id: 'one',
  title: 'A node',
  relation: 'child',
  x: 300,
  y: 0,
  width: 100,
  height: 36,
  order: 0,
  opacity: 1,
}

const getHandleOf = (options: { hasPill: boolean; viewportWidth?: number }) =>
  effectScope().run(
    () =>
      useOpenBox({
        node: () => node,
        expanded: () => null,
        on: () => null,
        delay: () => 500,
        hasPill: () => options.hasPill,
        viewportWidth: () => options.viewportWidth,
        clock: stubClock().clock,
      }).handle.value,
  )!

describe('the handle of a box without a pill', () => {
  it('moves to the leading side where the trailing side leaves the window', () => {
    // The box ends at 350, and its disc would end past half of 700.
    expect(getHandleOf({ hasPill: false, viewportWidth: 700 }).x).toBe(-(50 + HANDLE_RADIUS))
  })

  it('stays on the trailing side where the disc fits', () => {
    expect(getHandleOf({ hasPill: false, viewportWidth: 800 }).x).toBe(50 + HANDLE_RADIUS)
  })
})
