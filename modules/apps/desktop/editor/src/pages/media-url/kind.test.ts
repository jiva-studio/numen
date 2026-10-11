/**
 * How a url tab is reached, asked without a screen.
 *
 * A url that is not reachable from a path is a link that opens nothing.
 */
import { describe, expect, it, vi } from 'vitest'
import type { FileOpeners, SourceReader } from '@/entities/tab'
import { URLS } from './kind'

type RegisterReader = FileOpeners['registerReader']
type RegisterEditor = FileOpeners['registerEditor']

describe('the urls of the vault', () => {
  it('are read from what the vault says stands at a path', () => {
    const registerReader = vi.fn<RegisterReader>()
    const read = vi.fn<SourceReader>()

    const openers = { registerReader, registerEditor: vi.fn<RegisterEditor>() }
    URLS.register(openers as unknown as FileOpeners, read)

    expect(registerReader).toHaveBeenCalledExactlyOnceWith({ kind: 'url' }, read)
  })

  it('are read at a url just made, with no spans', () => {
    const registerEditor = vi.fn<RegisterEditor>()
    const read = vi.fn<SourceReader>()

    const openers = { registerReader: vi.fn<RegisterReader>(), registerEditor }
    URLS.register(openers as unknown as FileOpeners, read)
    const open = registerEditor.mock.calls[0]![1]
    open('links/Ants.url', '', 'here')

    expect(registerEditor.mock.calls[0]![0]).toBe('url')
    expect(read).toHaveBeenCalledExactlyOnceWith('links/Ants.url', [])
  })
})
