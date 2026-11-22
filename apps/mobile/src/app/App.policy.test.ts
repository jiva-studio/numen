/**
 * What the page asks for and draws, held against the policy it carries. The policy itself and the
 * hosts the sources may name are pinned by the check `ts/window-sources`. Nothing here runs on a
 * phone: what a WebView makes of the policy is answered by a device and by nothing else.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { NoteSheet } from '@/pages/note-sheet'
import { startCore } from '@/entities/core'
import type { Core } from '@/entities/core'

vi.mock('@capacitor/core', () => ({
  registerPlugin: () => ({ start: async () => ({ port: 45123, dir: '/data' }) }),
}))

afterEach(() => {
  vi.unstubAllGlobals()
  vi.resetModules()
  document.body.innerHTML = ''
})

describe('what the page asks for', () => {
  it('reaches the core on the loopback address and nowhere else', async () => {
    const request: string[] = []
    vi.stubGlobal('fetch', async (input: unknown) => {
      request.push(input instanceof Request ? input.url : String(input))
      throw new Error('this test serves nothing')
    })

    const core = await startCore()
    await expect(core.readNote('Physics.md')).rejects.toThrow()

    expect(request).not.toStrictEqual([])
    for (const address of request) expect(new URL(address).origin).toBe('http://127.0.0.1:45123')
  })
})

describe('a picture written into a note', () => {
  it('stays plain text on the page', async () => {
    const core = {
      readNote: vi.fn().mockResolvedValue({
        ok: true,
        value: {
          prose: 'A note synced from elsewhere.\n\n![](https://evil.example/pixel.png)\n',
          stamp: null,
        },
      }),
    } as unknown as Core

    const sheet = mount(NoteSheet, { props: { core, path: 'Synced.md' }, attachTo: document.body })
    await flushPromises()
    expect(sheet.get('textarea').element.value).toContain('https://evil.example/pixel.png')
    expect(document.querySelectorAll('img[src]')).toHaveLength(0)

    sheet.unmount()
  })
})
