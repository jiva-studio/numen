/**
 * What the page opens, and what it shows when the core cannot be reached.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { Code, ConnectError } from '@connectrpc/connect'
import { CapacitorException } from '@capacitor/core'
import { IonToast } from '@ionic/vue'
import { formatErrorMessage } from '@numen/core-client'
import VaultPage from './VaultPage.vue'
import NoteSheet from '../note/NoteSheet.vue'
import { startCore, type Core } from '../core'

vi.mock('../core', () => ({ startCore: vi.fn() }))

async function toastAfter(why: unknown) {
  vi.mocked(startCore).mockRejectedValueOnce(why)
  const page = mount(VaultPage)
  await flushPromises()
  const toast = page.getComponent(IonToast)
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

describe('a core that answers', () => {
  it('opens the opening note, and opens it again from its row once closed', async () => {
    const readNote = vi.fn().mockResolvedValue({ body: '# Physics\n', at: undefined })
    const getOpeningNote = vi.fn().mockResolvedValue({ note: { path: 'Physics.md' } })
    vi.mocked(startCore).mockResolvedValueOnce({
      notes: { readNote, getOpeningNote },
    } as unknown as Core)
    const page = mount(VaultPage, { attachTo: document.body })
    await flushPromises()

    expect(page.getComponent(NoteSheet).props('path')).toBe('Physics.md')
    expect(readNote).toHaveBeenCalledWith({ path: 'Physics.md' })

    page.getComponent(NoteSheet).vm.$emit('close')
    await flushPromises()
    expect(page.findComponent(NoteSheet).exists()).toBe(false)

    await page.get('[data-testid="note"]').trigger('click')
    await flushPromises()
    expect(page.getComponent(NoteSheet).props('path')).toBe('Physics.md')

    page.unmount()
  })

  it('opens no note and shows no row for a vault without an opening note', async () => {
    const getOpeningNote = vi.fn().mockResolvedValue({ note: undefined })
    vi.mocked(startCore).mockResolvedValueOnce({ notes: { getOpeningNote } } as unknown as Core)
    const page = mount(VaultPage, { attachTo: document.body })
    await flushPromises()

    expect(page.findComponent(NoteSheet).exists()).toBe(false)
    expect(page.find('[data-testid="note"]').exists()).toBe(false)

    page.unmount()
  })
})
