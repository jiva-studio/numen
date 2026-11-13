/** A note is settled when it is asked for, and once for a frame. */
import { nextTick, reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useSettledNodes } from './useSettledNodes'
import { arrangePlex } from '../lib/arrange'
import { build } from '../fixtures/build'

const frame = arrangePlex(build('Hub', { child: 3 }))

describe('the lazily settled notes of a frame', () => {
  it('compute nothing until a note is requested', () => {
    const widen = vi.fn(() => null)
    const hung = vi.fn(() => null)
    useSettledNodes(() => ({ nodes: frame.nodes, widen, hung }))

    expect(widen).not.toHaveBeenCalled()
    expect(hung).not.toHaveBeenCalled()
  })

  it('compute once for a note and a frame, and again when the frame changes', async () => {
    const hung = vi.fn(() => null)
    const source = reactive({ nodes: frame.nodes, hung })
    const { getHung } = useSettledNodes(() => source)
    const node = frame.nodes[1]!

    getHung(node)
    getHung(node)
    expect(hung).toHaveBeenCalledTimes(1)

    source.nodes = [...frame.nodes]
    await nextTick()
    getHung(node)
    expect(hung).toHaveBeenCalledTimes(2)
  })

  it('are nothing for a note when no sizes are given', () => {
    const { getWide, getHung } = useSettledNodes(() => ({ nodes: frame.nodes }))

    expect(getWide(frame.nodes[0]!)).toBeNull()
    expect(getHung(frame.nodes[0]!)).toBeNull()
  })
})
