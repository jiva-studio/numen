import { nextTick, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { runAfterRender } from './afterRender'

describe('runAfterRender', () => {
  it('runs the callback after the pending state is flushed, not before', async () => {
    const seen: string[] = []
    const state = ref('first')
    runAfterRender(() => seen.push(state.value))
    state.value = 'second'
    expect(seen).toEqual([])
    await nextTick()
    await nextTick()
    expect(seen).toEqual(['second'])
  })
})
