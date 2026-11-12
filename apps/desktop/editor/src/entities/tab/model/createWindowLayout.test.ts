import { describe, expect, it } from 'vitest'
import { createWindowLayout } from './createWindowLayout'

describe('the layout of a window', () => {
  it('is not ready and writes no tabs until it is started', () => {
    const layout = createWindowLayout(false)

    expect(layout.isReady.value).toBe(false)
    expect(layout.isWritingTabs.value).toBe(false)
  })

  it('writes the open tabs once started when it is the main window', () => {
    const layout = createWindowLayout(false)

    layout.markStarted()

    expect(layout.isReady.value).toBe(true)
    expect(layout.isWritingTabs.value).toBe(true)
  })

  it('never writes the open tabs when it was made for a detach', () => {
    const layout = createWindowLayout(true)

    layout.markStarted()

    expect(layout.isReady.value).toBe(true)
    expect(layout.isWritingTabs.value).toBe(false)
  })
})
