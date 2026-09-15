/**
 * What a note the core refused says on the phone.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ErrorCode } from '@numen/protocol'
import { formatErrorCodeMessage } from '@numen/wire'
import NoteSheet from './NoteSheet.vue'
import type { Core } from '../core'

const createCore = (notes: Record<string, unknown>) => ({ notes }) as unknown as Core

describe('a note the core refused', () => {
  it('says why it could not be read, in words, and closes', async () => {
    const core = createCore({
      readNote: vi.fn().mockResolvedValue({ body: '', at: undefined, error: ErrorCode.MISSING }),
    })
    const sheet = mount(NoteSheet, { props: { core, path: 'Gone.md' }, attachTo: document.body })
    await flushPromises()

    expect(sheet.emitted('error')).toStrictEqual([[formatErrorCodeMessage(ErrorCode.MISSING)]])
    expect(sheet.emitted('error')![0]![0]).not.toMatch(/\d/)
    expect(sheet.emitted('close')).toHaveLength(1)

    sheet.unmount()
  })

  it('says why it could not be written, in words, and stays open', async () => {
    const core = createCore({
      readNote: vi.fn().mockResolvedValue({ body: 'One line.\n', at: undefined }),
      writeNote: vi.fn().mockResolvedValue({ error: ErrorCode.STALE }),
    })
    const sheet = mount(NoteSheet, { props: { core, path: 'Kept.md' }, attachTo: document.body })
    await flushPromises()

    await sheet.get('[data-testid="keep"]').trigger('click')
    await flushPromises()

    expect(sheet.emitted('error')).toStrictEqual([[formatErrorCodeMessage(ErrorCode.STALE)]])
    expect(sheet.emitted('error')![0]![0]).not.toMatch(/\d/)
    expect(sheet.emitted('close')).toBeUndefined()

    sheet.unmount()
  })
})
