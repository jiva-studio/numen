/** What a note is settled to: computed once for a frame and the sizes, whatever else changes. */
import { nextTick, reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useSettledNodes } from './useSettledNodes'
import { arrangePlex } from '../lib/arrange'
import { build } from '../fixtures/build'

const frame = arrangePlex(build('Hub', { child: 3 }))

describe('the settled boxes of a frame', () => {
  it('settle each note once, and again only when the frame changes', async () => {
    const widen = vi.fn(() => null)
    const source = reactive({ nodes: frame.nodes, widen })
    const { wides } = useSettledNodes(() => source)

    const first = wides.value
    expect(wides.value).toBe(first)
    expect(widen).toHaveBeenCalledTimes(frame.nodes.length)

    source.nodes = frame.nodes.slice(1)
    await nextTick()

    expect(wides.value).not.toBe(first)
    expect([...wides.value.keys()]).toStrictEqual(frame.nodes.slice(1).map((node) => node.id))
  })

  it('are nothing for a note when no sizes are given', () => {
    const { wides, hungs } = useSettledNodes(() => ({ nodes: frame.nodes }))

    expect([...wides.value.values(), ...hungs.value.values()].every((one) => one === null)).toBe(
      true,
    )
  })
})
