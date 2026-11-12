/**
 * What the phone shows for a note the core rejected, and what it writes back.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { Core, NoteBaseline } from '@/entities/core'
import NoteSheet from './NoteSheet.vue'

const createCore = (calls: Partial<Pick<Core, 'readNote' | 'writeNote'>>) => calls as Core

const READ_FAILED = 'not found'
const WRITE_FAILED = 'changed elsewhere'

describe('a note the core rejected', () => {
  it('says why it could not be read, and closes', async () => {
    const core = createCore({
      readNote: vi.fn().mockResolvedValue({ ok: false, error: READ_FAILED }),
    })
    const sheet = mount(NoteSheet, { props: { core, path: 'Gone.md' }, attachTo: document.body })
    await flushPromises()

    expect(sheet.emitted('error')).toStrictEqual([[READ_FAILED]])
    expect(sheet.emitted('close')).toHaveLength(1)

    sheet.unmount()
  })

  it('says why it could not be written, and stays open', async () => {
    const core = createCore({
      readNote: vi
        .fn()
        .mockResolvedValue({ ok: true, value: { prose: 'One line.\n', stamp: null } }),
      writeNote: vi.fn().mockResolvedValue({ ok: false, error: WRITE_FAILED }),
    })
    const sheet = mount(NoteSheet, { props: { core, path: 'Kept.md' }, attachTo: document.body })
    await flushPromises()

    await sheet.get('[data-testid="keep"]').trigger('click')
    await flushPromises()

    expect(sheet.emitted('error')).toStrictEqual([[WRITE_FAILED]])
    expect(sheet.emitted('close')).toBeUndefined()

    sheet.unmount()
  })
})

describe('a note typed in on the phone', () => {
  it('writes the typed text against the version it read', async () => {
    const stamp: NoteBaseline['stamp'] = { path: 'Kept.md', size: 10n, mtime: 1n }
    const writeNote = vi.fn().mockResolvedValue({ ok: true, value: null })
    const core = createCore({
      readNote: vi.fn().mockResolvedValue({ ok: true, value: { prose: 'One line.\n', stamp } }),
      writeNote,
    })
    const sheet = mount(NoteSheet, { props: { core, path: 'Kept.md' }, attachTo: document.body })
    await flushPromises()

    expect(sheet.get('textarea').element.value).toBe('One line.\n')
    await sheet.get('textarea').setValue('Two lines.\n')
    await sheet.get('[data-testid="keep"]').trigger('click')
    await flushPromises()

    expect(writeNote).toHaveBeenCalledWith('Kept.md', 'Two lines.\n', {
      prose: 'One line.\n',
      stamp,
    })
    expect(sheet.emitted('close')).toHaveLength(1)

    sheet.unmount()
  })
})
