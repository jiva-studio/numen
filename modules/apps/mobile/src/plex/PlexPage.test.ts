/**
 * What the page says when the core cannot be reached.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { Code, ConnectError } from '@connectrpc/connect'
import { IonToast } from '@ionic/vue'
import { formatErrorMessage } from '@numen/wire'
import PlexPage from './PlexPage.vue'
import { startCore } from '../core'

vi.mock('../core', () => ({ startCore: vi.fn() }))

/** The toast the page puts its error on, after the core has answered. */
async function toastAfter(why: unknown) {
  vi.mocked(startCore).mockRejectedValueOnce(why)
  const page = mount(PlexPage)
  await flushPromises()
  const toast = page.getComponent(IonToast)
  return { page, open: toast.props('isOpen'), said: toast.props('message') }
}

describe('a core that cannot be reached', () => {
  it('says what the call returned, where the code carries a sentence', async () => {
    const caughtError = new ConnectError('numen is still starting', Code.Unavailable)
    const { page, open, said } = await toastAfter(caughtError)

    expect(open).toBe(true)
    expect(said).toBe('numen is still starting')

    page.unmount()
  })

  it('says what went wrong in words, whatever was thrown', async () => {
    const caughtError = new Error('ECONNREFUSED 127.0.0.1:45123')
    const { page, open, said } = await toastAfter(caughtError)

    expect(open).toBe(true)
    expect(said).toBe(formatErrorMessage(caughtError))
    expect(said).not.toContain('ECONNREFUSED')

    page.unmount()
  })
})
