/** The sizes of a note are computed when requested, and once for a frame. */
import { nextTick, reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useNodeSizes } from './useNodeSizes'
import { arrangePlex } from '../lib/arrange'
import { build } from '../fixtures/build'

const frame = arrangePlex(build('Hub', { child: 3 }))

describe('the lazily computed sizes of the notes of a frame', () => {
  it('compute nothing until a note is requested', () => {
    const widen = vi.fn(() => null)
    const placedHeadings = vi.fn(() => null)
    useNodeSizes(() => ({ nodes: frame.nodes, widen, placedHeadings }))

    expect(widen).not.toHaveBeenCalled()
    expect(placedHeadings).not.toHaveBeenCalled()
  })

  it('compute once for a note and a frame, and again when the frame changes', async () => {
    const placedHeadings = vi.fn(() => null)
    const source = reactive({ nodes: frame.nodes, placedHeadings })
    const { getPlacedHeadingsOf } = useNodeSizes(() => source)
    const node = frame.nodes[1]!

    getPlacedHeadingsOf(node)
    getPlacedHeadingsOf(node)
    expect(placedHeadings).toHaveBeenCalledTimes(1)

    source.nodes = [...frame.nodes]
    await nextTick()
    getPlacedHeadingsOf(node)
    expect(placedHeadings).toHaveBeenCalledTimes(2)
  })

  it('are nothing for a note when no sizes are given', () => {
    const { getExpandedBoxOf, getPlacedHeadingsOf } = useNodeSizes(() => ({ nodes: frame.nodes }))

    expect(getExpandedBoxOf(frame.nodes[0]!)).toBeNull()
    expect(getPlacedHeadingsOf(frame.nodes[0]!)).toBeNull()
  })
})
