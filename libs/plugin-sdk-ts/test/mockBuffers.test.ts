import { describe, expect, it, vi } from 'vitest'
import type { FileBuffers } from '../src/buffers'
import { createMockBufferHost } from '../src/testing/mockBuffers'

const createBuffers = (): FileBuffers => ({
  flush: vi.fn(async () => undefined),
})

describe('the mock buffer host', () => {
  it('holds the buffers handed over until they are release', () => {
    const host = createMockBufferHost()
    const buffers = createBuffers()

    const tracked = host.track(buffers)
    expect(host.getTracked()).toStrictEqual([buffers])

    tracked.dispose()
    expect(host.getTracked()).toStrictEqual([])
  })

  it('writes out the buffers and the handlers when the window quits', async () => {
    const host = createMockBufferHost()
    const buffers = createBuffers()
    const flushed = vi.spyOn(buffers, 'flush')
    const handler = vi.fn(async () => undefined)
    host.track(buffers)
    host.addFlushHandler(handler)

    await host.flush()

    expect(flushed).toHaveBeenCalledOnce()
    expect(handler).toHaveBeenCalledOnce()
  })

  it('writes out no handler that was release', async () => {
    const host = createMockBufferHost()
    const handler = vi.fn(async () => undefined)
    host.addFlushHandler(handler).dispose()

    await host.flush()

    expect(handler).not.toHaveBeenCalled()
  })
})
