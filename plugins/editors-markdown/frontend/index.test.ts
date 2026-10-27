import { describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import { activate, deactivate } from './index'

describe('editor-markdown plugin lifecycle', () => {
  it('registers markdown reader and note tab kind on activation', () => {
    const registerReader = vi.fn()
    const registerTabKind = vi.fn()
    const context = {
      registerReader,
      registerTabKind,
    } as unknown as PluginContext

    activate(context)

    expect(registerReader).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'editor-markdown',
        title: 'Markdown Note Editor',
        source: 'note',
      }),
    )

    expect(registerTabKind).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: 'note',
        title: 'Note',
      }),
    )
  })

  it('runs deactivation hook cleanly', () => {
    expect(() => deactivate()).not.toThrow()
  })
})
