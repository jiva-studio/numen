import { nextTick, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { runAfterRender } from './afterRender'

describe('runAfterRender', () => {
  it('runs the callback after the pending state is flushed, not before', async () => {
    const ran: string[] = []
    const state = ref('first')
    runAfterRender(() => ran.push(state.value))
    state.value = 'second'
    expect(ran).toEqual([])
    await nextTick()
    await nextTick()
    expect(ran).toEqual(['second'])
  })
})
