/**
 * What the window draws while the plugins of its interface start: the loading indicator alone, and
 * the whole window at once when they have started.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mountWindow } from '@/testing/window'

const start = vi.hoisted(() => ({ isStarted: { value: false } }))

vi.mock('./bootstrap/useAppBootstrap', async () => {
  const { ref } = await import('vue')
  const isStarted = ref(false)
  start.isStarted = isStarted
  return {
    STEP_LIMIT_MS: 10_000,
    useAppBootstrap: () => ({ failure: ref(''), isStarted, retry: vi.fn() }),
  }
})

describe('the window while the plugins of its interface start', () => {
  it('shows only the loading indicator', async () => {
    start.isStarted.value = false

    const window = await mountWindow()

    expect(window.find('[role="status"][aria-busy="true"]').exists()).toBe(true)
    expect(window.find('main').exists()).toBe(false)
    expect(window.find('[data-workspace-strip]').exists()).toBe(false)
  })

  it('draws the whole window at once when they have started', async () => {
    start.isStarted.value = false
    const window = await mountWindow()

    start.isStarted.value = true
    await nextTick()

    expect(window.find('main').exists()).toBe(true)
    expect(window.find('[role="status"][aria-busy="true"]').exists()).toBe(false)
  })
})
