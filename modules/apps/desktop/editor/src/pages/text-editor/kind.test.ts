/**
 * The settings file's tab: one to a window, and the way to put it in front.
 */
import { describe, expect, it, vi } from 'vitest'
import { createTextEditorTabKind } from './kind'
import type { TextEditorTabDeps } from './model/useTextEditor'
import { WORDS as words } from './words'
import { SETTINGS_FILE } from '@/entities/tab'
import type { WindowHandle } from '@/entities/tab'

const createHandle = (): WindowHandle => ({
  openTab: vi.fn<WindowHandle['openTab']>(() => Promise.resolve('tab')),
  openTabBeside: vi.fn<WindowHandle['openTabBeside']>(() => Promise.resolve('tab')),
  show: vi.fn<WindowHandle['show']>(),
  closeTab: vi.fn<WindowHandle['closeTab']>(),
  each: () => [],
  last: () => null,
  front: vi.fn<WindowHandle['front']>(() => null),
  getTabState: () => null,
})

const createCore = (): TextEditorTabDeps => ({
  getSettingsFile: () => Promise.resolve({ written: '{}\n', path: '/numen.json' }),
  saveSettingsFile: () => Promise.resolve({ isChanged: false }),
})

describe('the settings file tab', () => {
  it('is called what the file is called', () => {
    const { kind } = createTextEditorTabKind(createHandle(), createCore(), vi.fn())
    const state = kind.open('')

    expect(kind.getTitle?.(state)).toBe(words.called)
  })

  it('is one tab, whatever it is opened on', () => {
    const { kind } = createTextEditorTabKind(createHandle(), createCore(), vi.fn())

    expect(kind.kind).toBe(SETTINGS_FILE)
    expect(kind.identity?.('anything')).toBe(SETTINGS_FILE)
    expect(kind.identity?.('something else')).toBe(SETTINGS_FILE)
  })

  it('carries a dot where what is typed differs from the file', async () => {
    const { kind } = createTextEditorTabKind(createHandle(), createCore(), vi.fn())
    const state = kind.open('')
    await state.reload()

    expect(kind.getMark?.(state)).toBeUndefined()
    state.type('{ }\n')
    expect(kind.getMark?.(state)).toBe('•')
  })

  it('carries the word stale where the file moved past what was read', async () => {
    const core: TextEditorTabDeps = {
      ...createCore(),
      saveSettingsFile: () => Promise.resolve({ isChanged: true }),
    }
    const { kind } = createTextEditorTabKind(createHandle(), core, vi.fn())
    const state = kind.open('')
    await state.reload()
    state.type('{ }\n')
    await state.save()

    expect(kind.getMark?.(state)).toBe('stale')
  })

  it('is put in front beside the pane the person is in', () => {
    const handle = createHandle()
    const { openSettingsFile } = createTextEditorTabKind(handle, createCore(), vi.fn())

    openSettingsFile()

    expect(handle.openTabBeside).toHaveBeenCalledTimes(1)
    expect(handle.openTabBeside).toHaveBeenCalledWith(SETTINGS_FILE)
  })
})
