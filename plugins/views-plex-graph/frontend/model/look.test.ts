import { describe, expect, it } from 'vitest'
import type { JsonValue } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { readLook } from './look'

const createContext = (initialSettings: Record<string, JsonValue> = {}) =>
  createMockPluginContext({ pluginId: 'md.numen.views.plex.graph', initialSettings })

/** The look of a plex as the manifest declares its defaults. */
const DEFAULTS = {
  showPills: true,
  pillPaddingSides: 10,
  pillPaddingEnds: 9.25,
  focusSize: 1.22,
  animationLength: 420,
  lineThickness: 2,
  outlineThickness: 1,
  lineCurvature: 0.55,
  noteRoom: 0,
  namesOpacity: 0.7,
  focusedNamesOpacity: 1,
  references: 'both',
}

/** A value for every setting that is none of the defaults, and inside every range. */
const CHOSEN = {
  showPills: false,
  pillPaddingSides: 4,
  pillPaddingEnds: 3.5,
  focusSize: 1.6,
  animationLength: 900,
  lineThickness: 3,
  outlineThickness: 1.5,
  lineCurvature: 0.2,
  noteRoom: 12,
  namesOpacity: 0.2,
  focusedNamesOpacity: 0.8,
  references: 'outgoing',
}

const BELOW = {
  pillPaddingSides: -5,
  pillPaddingEnds: -1,
  focusSize: 0.5,
  animationLength: -100,
  lineThickness: 0,
  outlineThickness: 0.1,
  lineCurvature: -0.4,
  noteRoom: -10,
  namesOpacity: -1,
  focusedNamesOpacity: -0.5,
}

const ABOVE = {
  pillPaddingSides: 99,
  pillPaddingEnds: 40,
  focusSize: 9,
  animationLength: 10_000,
  lineThickness: 12,
  outlineThickness: 9,
  lineCurvature: 3,
  noteRoom: 400,
  namesOpacity: 3,
  focusedNamesOpacity: 2,
}

const LOWEST = {
  pillPaddingSides: 0,
  pillPaddingEnds: 0,
  focusSize: 1,
  animationLength: 0,
  lineThickness: 0.5,
  outlineThickness: 0.5,
  lineCurvature: 0,
  noteRoom: 0,
  namesOpacity: 0,
  focusedNamesOpacity: 0.05,
}

const HIGHEST = {
  pillPaddingSides: 24,
  pillPaddingEnds: 16,
  focusSize: 2.5,
  animationLength: 2000,
  lineThickness: 4,
  outlineThickness: 4,
  lineCurvature: 1,
  noteRoom: 40,
  namesOpacity: 1,
  focusedNamesOpacity: 1,
}

type Look = ReturnType<typeof readLook>

const getValues = (look: Look, keys: readonly string[]): Record<string, unknown> =>
  Object.fromEntries(
    keys.map((key) => [key, (look as unknown as Record<string, { value: unknown }>)[key]?.value]),
  )

describe('readLook', () => {
  it('never lets the names in focus vanish, and lets the resting names', () => {
    const look = readLook(createContext({ namesOpacity: 0, focusedNamesOpacity: 0 }))

    expect(look.namesOpacity.value).toBe(0)
    expect(look.focusedNamesOpacity.value).toBe(0.05)
  })

  it('draws the default look where nothing is set', () => {
    const look = readLook(createContext())

    expect(getValues(look, Object.keys(DEFAULTS))).toStrictEqual(DEFAULTS)
  })

  it('reads every value the settings hold, under its own key', () => {
    const look = readLook(createContext(CHOSEN))

    expect(getValues(look, Object.keys(CHOSEN))).toStrictEqual(CHOSEN)
  })

  it('takes the default for a value of the wrong kind', () => {
    const asText = Object.fromEntries(Object.keys(DEFAULTS).map((key) => [key, 'seven']))
    const look = readLook(createContext(asText))

    expect(getValues(look, Object.keys(DEFAULTS))).toStrictEqual(DEFAULTS)
  })

  it('takes the default for a number that is not finite', () => {
    const look = readLook(createContext({ focusSize: null, lineThickness: [] }))

    expect(look.focusSize.value).toBe(1.22)
    expect(look.lineThickness.value).toBe(2)
  })

  it('clamps a number below its range to the lowest value', () => {
    const look = readLook(createContext(BELOW))

    expect(getValues(look, Object.keys(LOWEST))).toStrictEqual(LOWEST)
  })

  it('clamps a number above its range to the highest value', () => {
    const look = readLook(createContext(ABOVE))

    expect(getValues(look, Object.keys(HIGHEST))).toStrictEqual(HIGHEST)
  })

  it('follows every setting the person changes', () => {
    const context = createContext()
    const look = readLook(context)

    for (const [key, value] of Object.entries(CHOSEN)) context.setSetting(key, value)

    expect(getValues(look, Object.keys(CHOSEN))).toStrictEqual(CHOSEN)
  })

  it('repairs a changed value as it repairs a stored one', () => {
    const context = createContext(CHOSEN)
    const look = readLook(context)

    context.setSetting('focusSize', 7)
    context.setSetting('lineCurvature', 'straight')
    context.setSetting('showPills', 'no')

    expect(look.focusSize.value).toBe(2.5)
    expect(look.lineCurvature.value).toBe(0.55)
    expect(look.showPills.value).toBe(true)
  })

  it('stops following the settings once the plugin is disposed', () => {
    const context = createContext()
    const look = readLook(context)

    context.dispose()
    context.setSetting('showPills', false)
    context.setSetting('noteRoom', 20)

    expect(look.showPills.value).toBe(true)
    expect(look.noteRoom.value).toBe(0)
  })
})

describe('the References setting', () => {
  it.each(['off', 'outgoing', 'incoming', 'both'])('reads %s', (mode) => {
    expect(readLook(createContext({ references: mode })).references.value).toBe(mode)
  })

  it.each([['seven'], [3], [null], [true]])('falls back to both for %j', (stored) => {
    const look = readLook(createContext({ references: stored as JsonValue }))

    expect(look.references.value).toBe('both')
  })

  it('follows the person changing it and repairs a changed value to both', () => {
    const context = createContext()
    const look = readLook(context)

    context.setSetting('references', 'incoming')
    expect(look.references.value).toBe('incoming')

    context.setSetting('references', 'sometimes')
    expect(look.references.value).toBe('both')
  })
})
