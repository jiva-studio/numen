/** What the drawing attends to: set by a note, and cleared when that note leaves the frame. */
import { nextTick, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useAttention } from './useAttention'
import { arrangePlex } from '../lib/arrange'
import { build } from '../fixtures/build'

const whole = arrangePlex(build('Hub', { child: 3 }))
const [first, second] = whole.nodes
  .filter((node) => node.relation === 'child')
  .map((node) => node.id) as [string, string]
const without = (id: string) => ({ ...whole, nodes: whole.nodes.filter((node) => node.id !== id) })

describe('the attention of the drawing', () => {
  it('is released by the note that took it, and not by another', () => {
    const { attention, setAttention } = useAttention(() => whole)
    setAttention(first, true)

    setAttention(second, false)
    expect(attention.node).toBe(first)

    setAttention(first, false)
    expect(attention.node).toBeNull()
  })

  it('is cleared when the note it rests on leaves the frame', async () => {
    const frame = ref(whole)
    const { attention, setAttention } = useAttention(() => frame.value)
    setAttention(first, true)

    frame.value = without(first)
    await nextTick()

    expect(attention.node).toBeNull()
  })

  it('stays on a note that is still in the frame', async () => {
    const frame = ref(whole)
    const { attention, setAttention } = useAttention(() => frame.value)
    setAttention(second, true)

    frame.value = without(first)
    await nextTick()

    expect(attention.node).toBe(second)
  })
})
