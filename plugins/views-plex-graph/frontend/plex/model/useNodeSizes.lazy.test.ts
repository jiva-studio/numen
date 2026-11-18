/** The expanded box of a note is computed when requested, and once for a frame. */
import { nextTick, reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useNodeSizes } from './useNodeSizes'
import { arrangePlex } from '../lib/arrange'
import { build } from '../fixtures/build'

const frame = arrangePlex(build('Hub', { child: 3 }))

describe('the lazily computed sizes of the notes of a frame', () => {
  it('compute nothing until a note is requested', () => {
    const widen = vi.fn(() => null)
    useNodeSizes(() => ({ nodes: frame.nodes, widen }))

    expect(widen).not.toHaveBeenCalled()
  })

  it('compute once for a note and a frame, and again when the frame changes', async () => {
    const widen = vi.fn(() => null)
    const source = reactive({ nodes: frame.nodes, widen })
    const { getExpandedBoxOf } = useNodeSizes(() => source)
    const node = frame.nodes[1]!

    getExpandedBoxOf(node)
    getExpandedBoxOf(node)
    expect(widen).toHaveBeenCalledTimes(1)

    source.nodes = [...frame.nodes]
    await nextTick()
    getExpandedBoxOf(node)
    expect(widen).toHaveBeenCalledTimes(2)
  })

  it('are nothing for a note when no sizes are given', () => {
    const { getExpandedBoxOf } = useNodeSizes(() => ({ nodes: frame.nodes }))

    expect(getExpandedBoxOf(frame.nodes[0]!)).toBeNull()
  })
})
