import { describe, expect, it, vi } from 'vitest'
import type { JsonValue, PluginContext } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { NAMES_PER_NOTE, SHOW_NAMES, readNames } from './names'

const createContext = (initialSettings: Record<string, JsonValue> = {}) =>
  createMockPluginContext({ pluginId: 'md.numen.views.plex.graph', initialSettings })

describe('readNames', () => {
  it('shows six names where nothing is set', () => {
    const { showNames, namesPerNote } = readNames(createContext())

    expect(SHOW_NAMES).toBe(true)
    expect(NAMES_PER_NOTE).toBe(6)
    expect(showNames.value).toBe(true)
    expect(namesPerNote.value).toBe(6)
  })

  it('reads the values from the camelCase keys', () => {
    const { showNames, namesPerNote } = readNames(
      createContext({ showNames: false, namesPerNote: 3 }),
    )

    expect(showNames.value).toBe(false)
    expect(namesPerNote.value).toBe(3)
  })

  it('ignores the snake_case keys of the same settings', () => {
    const { showNames, namesPerNote } = readNames(
      createContext({ show_names: false, names_per_note: 3 }),
    )

    expect(showNames.value).toBe(true)
    expect(namesPerNote.value).toBe(6)
  })

  it('follows a setting the person changes', () => {
    const context = createContext()
    const { showNames, namesPerNote } = readNames(context)

    context.setSetting('showNames', false)
    context.setSetting('namesPerNote', 9)

    expect(showNames.value).toBe(false)
    expect(namesPerNote.value).toBe(9)
  })

  it('clamps a count outside the range and rounds one that is no integer', () => {
    const context = createContext({ namesPerNote: 24 })
    const { namesPerNote } = readNames(context)
    expect(namesPerNote.value).toBe(12)

    context.setSetting('namesPerNote', 0)
    expect(namesPerNote.value).toBe(1)
    context.setSetting('namesPerNote', 2.4)
    expect(namesPerNote.value).toBe(2)
    context.setSetting('namesPerNote', 2.5)
    expect(namesPerNote.value).toBe(3)
  })

  it('takes the default for a value that is no number', () => {
    const context = createContext({ namesPerNote: '4' })
    const { namesPerNote } = readNames(context)
    expect(namesPerNote.value).toBe(6)

    context.setSetting('namesPerNote', null)
    expect(namesPerNote.value).toBe(6)
  })

  it('reads the keys camelCase through a context that holds only snake_case values', () => {
    const get = vi.fn(() => undefined)
    readNames({ settings: { get, onChange: () => ({ dispose: vi.fn() }) } } as unknown as Pick<
      PluginContext,
      'settings'
    >)

    expect(get).toHaveBeenCalledWith('showNames')
    expect(get).toHaveBeenCalledWith('namesPerNote')
  })
})
