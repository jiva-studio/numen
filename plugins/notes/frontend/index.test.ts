import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createNamedIcon } from '@numen/ui'
import type { Component } from 'vue'
import { NOTE_EDITOR, type HostServices } from '@numen/plugin-sdk'
import { activate, deactivate } from './index'
import { createEditorSettingsContext } from './testing/editorSettings'

/** Host services whose vault offers no notes. */
const NO_NOTES = { vault: {}, ui: {} } as unknown as HostServices

const ICON_LOAD_TIMEOUT = 30_000

const drawIcon = (icon: Component): Promise<string> => {
  const wrapper = mount(icon)
  return vi.waitFor(
    () => {
      const svg = wrapper.find('svg')
      expect(svg.exists()).toBe(true)
      return svg.element.innerHTML
    },
    { timeout: ICON_LOAD_TIMEOUT, interval: 50 },
  )
}

describe('the icon of the notes in the tab strip', () => {
  const getRegisteredIcons = (): { reader: Component; tabKind: Component } => {
    const { context } = createEditorSettingsContext()
    activate(context, NO_NOTES)
    const reader = vi.mocked(context.register.reader).mock.calls[0]?.[0].icon
    const tabKind = vi.mocked(context.register.tabKind).mock.calls[0]?.[0].icon
    if (!reader || !tabKind)
      throw new Error('the plugin registered a reader or a tab kind without an icon')
    return { reader, tabKind }
  }

  it(
    'is drawn for the note tab kind as the file-text icon the manifest names',
    async () => {
      const { tabKind } = getRegisteredIcons()

      expect(await drawIcon(tabKind)).toBe(await drawIcon(createNamedIcon('file-text')))
    },
    ICON_LOAD_TIMEOUT * 1.5,
  )

  it(
    'is drawn for the markdown reader as the file-text icon the manifest names',
    async () => {
      const { reader } = getRegisteredIcons()

      expect(await drawIcon(reader)).toBe(await drawIcon(createNamedIcon('file-text')))
    },
    ICON_LOAD_TIMEOUT * 1.5,
  )
})

describe('notes plugin lifecycle', () => {
  it('registers markdown reader and note tab kind on activation', () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })

    activate(context, NO_NOTES)

    expect(context.register.reader).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'markdown',
        title: 'Markdown Note Editor',
      }),
    )

    expect(context.register.tabKind).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: 'note',
        title: 'Note',
      }),
    )
  })

  it('registers markdown reader and note tab kind before the editor settings have arrived', () => {
    const { context } = createEditorSettingsContext(undefined)

    activate(context, NO_NOTES)

    expect(context.register.reader).toHaveBeenCalledOnce()
    expect(context.register.tabKind).toHaveBeenCalledOnce()
  })

  it('registers no command, since creating a note is a command of the editor', () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })

    activate(context, NO_NOTES)

    expect(context.register.command).not.toHaveBeenCalled()
  })

  it('registers the note-editor view where the host offers the notes of the vault', () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })
    const host = {
      vault: { notes: { read: vi.fn(), write: vi.fn() } },
      ui: {},
    } as unknown as HostServices

    activate(context, host)

    expect(context.register.view).toHaveBeenCalledWith(
      expect.objectContaining({ capability: NOTE_EDITOR }),
    )
  })

  it('runs deactivation hook cleanly', () => {
    expect(() => deactivate()).not.toThrow()
  })
})
