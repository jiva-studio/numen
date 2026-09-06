/**
 * A link in a note, on a phone.
 *
 * This window has no address bar either, and a note may have been synced from
 * anywhere. What the phone draws of a note is the text itself, so no address a
 * note writes becomes one the WebView could be sent to. Drawn as prose the same
 * note gives three, which is what this holds the phone away from.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import NoteSheet from './NoteSheet.vue'
import type { Core } from '../core'

/** Every way a note writes an address, in one note. */
const WRITTEN = [
  '[press](https://evil.example/marked)',
  '<https://evil.example/angled>',
  'https://evil.example/bare',
  '<a href="https://evil.example/raw">raw</a>',
  '<img src="https://evil.example/pixel.png">',
  '[[a note]]',
].join('\n\n')

const holding = (body: string) =>
  ({
    notes: { readNote: vi.fn().mockResolvedValue({ body, at: undefined }) },
  }) as unknown as Core

describe('a note read on the phone', () => {
  it('puts no address in the page for the window to be sent to', async () => {
    const sheet = mount(NoteSheet, {
      props: { core: holding(WRITTEN), path: 'Synced.md' },
      attachTo: document.body,
    })
    await flushPromises()

    // The note is drawn, and every address in it is drawn as the words it is.
    expect(document.body.innerHTML).toContain('evil.example')
    const reaching = document.querySelectorAll('a[href], area[href], img[src], form[action]')
    expect([...reaching].map((each) => each.outerHTML)).toStrictEqual([])

    sheet.unmount()
  })
})
