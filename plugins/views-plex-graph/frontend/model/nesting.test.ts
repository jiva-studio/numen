import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import {
  DEFAULT_HEADING_LIMIT,
  HEADING_LIMIT,
  HEADING_LIMIT_MAX,
  HEADING_LIMIT_MIN,
  NEST_HEADINGS,
  readNesting,
} from './nesting'

const createContext = (initialSettings: Record<string, unknown> = {}) =>
  createMockPluginContext({ pluginId: 'md.numen.views.plex.graph', initialSettings })

describe('readNesting', () => {
  it('nests the default number of headings where nothing is set', () => {
    const { isNesting, headings } = readNesting(createContext())

    expect(isNesting.value).toBe(true)
    expect(headings.value).toBe(DEFAULT_HEADING_LIMIT)
  })

  it('reads the values the settings hold', () => {
    const { isNesting, headings } = readNesting(
      createContext({ [NEST_HEADINGS]: false, [HEADING_LIMIT]: 3 }),
    )

    expect(isNesting.value).toBe(false)
    expect(headings.value).toBe(3)
  })

  it('follows a setting the person changes', () => {
    const context = createContext()
    const { isNesting, headings } = readNesting(context)

    context.setSetting(NEST_HEADINGS, false)
    context.setSetting(HEADING_LIMIT, 9)

    expect(isNesting.value).toBe(false)
    expect(headings.value).toBe(9)
  })

  it('clamps a limit outside the range and takes the default for one that is no whole number', () => {
    const context = createContext({ [HEADING_LIMIT]: 40 })
    const { headings } = readNesting(context)
    expect(headings.value).toBe(HEADING_LIMIT_MAX)

    context.setSetting(HEADING_LIMIT, 0)
    expect(headings.value).toBe(HEADING_LIMIT_MIN)

    context.setSetting(HEADING_LIMIT, '4')
    expect(headings.value).toBe(DEFAULT_HEADING_LIMIT)
  })

  it('stops following the settings once the plugin is disposed', () => {
    const context = createContext()
    const { isNesting } = readNesting(context)

    context.dispose()
    context.setSetting(NEST_HEADINGS, false)

    expect(isNesting.value).toBe(true)
  })
})

describe('the manifest settings', () => {
  const { settings } = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../../manifest.json'), 'utf8'),
  ) as { settings: readonly Record<string, unknown>[] }

  it('declares the two keys with the defaults and range the plugin reads', () => {
    expect(settings.find((one) => one.key === NEST_HEADINGS)).toMatchObject({
      type: 'boolean',
      default: true,
      palette: true,
    })
    expect(settings.find((one) => one.key === HEADING_LIMIT)).toMatchObject({
      type: 'number',
      default: DEFAULT_HEADING_LIMIT,
      min: HEADING_LIMIT_MIN,
      max: HEADING_LIMIT_MAX,
    })
  })
})
