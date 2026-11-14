/**
 * What the page shows when the core cannot be reached.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { Code, ConnectError } from '@connectrpc/connect'
import type * as Capacitor from '@capacitor/core'
import { CapacitorException } from '@capacitor/core'
import { IonToast } from '@ionic/vue'
import { formatErrorMessage } from '@numen/core-client'
import VaultPage from './VaultPage.vue'

const plugin = vi.hoisted(() => ({ start: vi.fn() }))

vi.mock('@capacitor/core', async (importOriginal) => ({
  ...(await importOriginal<typeof Capacitor>()),
  registerPlugin: () => plugin,
}))

async function toastAfter(why: unknown) {
  plugin.start.mockRejectedValueOnce(why)
  const page = mount(VaultPage)
  const toast = page.getComponent(IonToast)
  expect(toast.props('isOpen')).toBe(false)
  expect(toast.props('message')).toBe('')
  await flushPromises()
  return { page, open: toast.props('isOpen'), message: toast.props('message') }
}

describe('a core that cannot be reached', () => {
  it('says what the call returned, where the code carries a sentence', async () => {
    const caughtError = new ConnectError('numen is still starting', Code.Unavailable)
    const { page, open, message } = await toastAfter(caughtError)

    expect(open).toBe(true)
    expect(message).toBe('numen is still starting')

    page.unmount()
  })

  it('says what went wrong in words, whatever was thrown', async () => {
    const caughtError = new Error('ECONNREFUSED 127.0.0.1:45123')
    const { page, open, message } = await toastAfter(caughtError)

    expect(open).toBe(true)
    expect(message).toBe(formatErrorMessage(caughtError))
    expect(message).not.toContain('ECONNREFUSED')

    page.unmount()
  })

  it('shows the generic text for a rejected native start', async () => {
    const caughtError = new CapacitorException(
      'mkdir /data/user/0/md.numen.sample/files/vault: permission denied',
    )
    const { page, open, message } = await toastAfter(caughtError)

    expect(open).toBe(true)
    expect(message).toBe('Numen is not responding; retrying')

    page.unmount()
  })
})
