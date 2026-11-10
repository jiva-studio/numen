/**
 * The session calls of the window core, over the real clients and a router in place of the core.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreHandlers } from '@numen/core-client/testing'
import { core } from './index'

const request = {
  writeOpenTabs: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  restore = installCoreHandlers(request)
})

afterEach(() => {
  restore()
})

describe('session domain', () => {
  it('writes the open tabs', async () => {
    request.writeOpenTabs.mockResolvedValue({})
    await core.writeOpenTabs({ tabs: [], activeId: '' })
    expect(request.writeOpenTabs).toHaveBeenCalled()
  })
})
