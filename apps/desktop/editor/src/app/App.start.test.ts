/**
 * What the window draws while the plugins of its interface start: the loading indicator alone, and
 * the whole window once they have started.
 */
import { describe, expect, it } from 'vitest'
import { mountStartingWindow, settle } from '@/testing/window'

describe('the window while the plugins of its interface start', () => {
  it('shows only the loading indicator', () => {
    const window = mountStartingWindow()

    expect(window.find('[role="status"][aria-busy="true"]').exists()).toBe(true)
    expect(window.find('main').exists()).toBe(false)
  })

  it('draws the whole window at once when they have started', async () => {
    const window = mountStartingWindow()

    await settle()
    await settle()
    await new Promise((isDone) => setTimeout(isDone, 50))

    expect(window.find('main').exists()).toBe(true)
    expect(window.find('[role="status"][aria-busy="true"]').exists()).toBe(false)
  })
})
