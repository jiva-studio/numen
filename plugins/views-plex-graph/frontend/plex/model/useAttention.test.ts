/** What the drawing attends to: set by a note or its names, and cleared when that note leaves the frame. */
import { nextTick, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useAttention } from './useAttention'
import { arrangePlex } from '../lib/arrange'
import { build } from '../fixtures/build'

const whole = arrangePlex(build('Hub', { child: 3 }))
const without = (id: string) => ({ ...whole, nodes: whole.nodes.filter((node) => node.id !== id) })

describe('the attention of the drawing', () => {
  it('rests on the note that took it, and on the names of another at once', () => {
    const { attention, attend } = useAttention(() => whole)

    attend('node', 'focus/child-0', true)
    attend('names', 'focus/child-1', true)

    expect({ ...attention }).toStrictEqual({ node: 'focus/child-0', names: 'focus/child-1' })
  })

  it('is released by the note that took it, and not by another', () => {
    const { attention, attend } = useAttention(() => whole)
    attend('node', 'focus/child-0', true)

    attend('node', 'focus/child-1', false)
    expect(attention.node).toBe('focus/child-0')

    attend('node', 'focus/child-0', false)
    expect(attention.node).toBeNull()
  })

  it('is cleared when the note it rests on leaves the frame', async () => {
    const frame = ref(whole)
    const { attention, attend } = useAttention(() => frame.value)
    attend('node', 'focus/child-0', true)
    attend('names', 'focus/child-0', true)

    frame.value = without('focus/child-0')
    await nextTick()

    expect({ ...attention }).toStrictEqual({ node: null, names: null })
  })

  it('stays on a note that is still in the frame', async () => {
    const frame = ref(whole)
    const { attention, attend } = useAttention(() => frame.value)
    attend('node', 'focus/child-1', true)

    frame.value = without('focus/child-0')
    await nextTick()

    expect(attention.node).toBe('focus/child-1')
  })
})
