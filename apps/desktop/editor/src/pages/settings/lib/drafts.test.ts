import { describe, expect, it, vi } from 'vitest'
import { createTextDrafts } from './drafts'

describe('createTextDrafts', () => {
  it('holds the text typed at a key until it is written', () => {
    const write = vi.fn()
    const text = createTextDrafts(write)
    text.setDraft('endpoint', 'http://127.0.0.1:9/v1')
    expect(text.getDraft('endpoint')).toBe('http://127.0.0.1:9/v1')
    expect(text.getDraft('model')).toBeUndefined()

    text.writeDraft('endpoint', 'http://127.0.0.1:9/v1')
    expect(write).toHaveBeenCalledWith('endpoint', 'http://127.0.0.1:9/v1')
    expect(text.getDraft('endpoint')).toBeUndefined()
  })
})
