import { describe, expect, it, vi } from 'vitest'
import { appendPathEntry, useTextDrafts, removePathEntry } from './drafts'

describe('useTextDrafts', () => {
  it('holds the text typed at a key until it is written and preserves other keys', () => {
    const write = vi.fn()
    const text = useTextDrafts(write)
    text.setDraft('endpoint', 'http://127.0.0.1:9/v1')
    text.setDraft('model', 'gpt-4')
    expect(text.getDraft('endpoint')).toBe('http://127.0.0.1:9/v1')
    expect(text.getDraft('model')).toBe('gpt-4')
    expect(text.getDraft('other')).toBeUndefined()

    text.writeDraft('endpoint', 'http://127.0.0.1:9/v1')
    expect(write).toHaveBeenCalledWith('endpoint', 'http://127.0.0.1:9/v1')
    expect(text.getDraft('endpoint')).toBeUndefined()
    expect(text.getDraft('model')).toBe('gpt-4')
  })
})

describe('appendPathEntry and removePathEntry', () => {
  it('appends non-empty unique paths and trims existing entries', () => {
    expect(appendPathEntry('/opt/a', '')).toBe('/opt/a')
    expect(appendPathEntry('', '/opt/a')).toBe('/opt/a')
    expect(appendPathEntry('/opt/a', '/opt/a')).toBe('/opt/a')
    expect(appendPathEntry('  /opt/a \n\n /opt/b  ', '/opt/b')).toBe('/opt/a\n/opt/b')
    expect(appendPathEntry('  /opt/a \n\n /opt/b  ', '/opt/c')).toBe('/opt/a\n/opt/b\n/opt/c')
  })

  it('removes a specific path or clears all when targetPath is empty or omitted', () => {
    expect(removePathEntry('/opt/a\n/opt/b')).toBe('')
    expect(removePathEntry('/opt/a\n/opt/b', '')).toBe('')
    expect(removePathEntry('  /opt/a \n\n /opt/b  ', '/opt/a')).toBe('/opt/b')
    expect(removePathEntry('/opt/a\n/opt/b\n/opt/c', '/opt/b')).toBe('/opt/a\n/opt/c')
  })
})
