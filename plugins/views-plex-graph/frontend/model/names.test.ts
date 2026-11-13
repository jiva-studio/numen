import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
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

  it('takes the default for a value outside the range or no number', () => {
    const context = createContext({ namesPerNote: 40 })
    const { namesPerNote } = readNames(context)
    expect(namesPerNote.value).toBe(6)

    context.setSetting('namesPerNote', 0)
    expect(namesPerNote.value).toBe(6)
    context.setSetting('namesPerNote', '4')
    expect(namesPerNote.value).toBe(6)
    context.setSetting('namesPerNote', 12)
    expect(namesPerNote.value).toBe(12)
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

describe('the manifest settings of the names', () => {
  const { settings } = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../../manifest.json'), 'utf8'),
  ) as { settings: readonly Record<string, unknown>[] }

  it('declares whether to show names, on by default', () => {
    expect(settings.find((one) => one.key === 'showNames')).toMatchObject({
      type: 'boolean',
      title: 'Show notes related to related notes',
      default: true,
      palette: true,
    })
  })

  it('declares how many names a note shows, from 1 to 12, six by default', () => {
    expect(settings.find((one) => one.key === 'namesPerNote')).toMatchObject({
      type: 'number',
      title: 'Names per note',
      default: 6,
      min: 1,
      max: 12,
      step: 1,
    })
  })

  it('places both after the headings limit', () => {
    const keys = settings.map((one) => one.key)

    expect(keys.indexOf('showNames')).toBeGreaterThan(keys.indexOf('headingLimit'))
    expect(keys.indexOf('namesPerNote')).toBeGreaterThan(keys.indexOf('headingLimit'))
  })
})
