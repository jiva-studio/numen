/** The two settings as the code reads them: the keys the manifest declares, their defaults and their range. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import type { JsonValue } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { readSettings } from './settings'

interface Declared {
  readonly key: string
  readonly default: unknown
  readonly min?: number
  readonly max?: number
}

const { settings: declared } = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../../manifest.json'), 'utf8'),
) as { settings: readonly Declared[] }

const read = (initialSettings: Record<string, JsonValue> = {}) => {
  const context = createMockPluginContext({
    pluginId: 'md.numen.views.plex.headings',
    initialSettings,
  })
  return { context, settings: readSettings(context) }
}

describe('the settings of the headings', () => {
  it('start at the default the manifest declares', () => {
    const { settings } = read()

    expect([settings.isNesting.value, settings.headingLimit.value]).toStrictEqual(
      ['nestHeadings', 'headingLimit'].map((key) => declared.find((one) => one.key === key)?.default),
    )
  })

  it('are read under the keys the manifest declares and no other', () => {
    const { settings } = read({ nest_headings: false, heading_limit: 3 })

    expect(settings.isNesting.value).toBe(true)
    expect(settings.headingLimit.value).toBe(6)
  })

  it('take the limit the manifest declares and no value beyond it', () => {
    const range = declared.find((one) => one.key === 'headingLimit')!

    expect(read({ headingLimit: range.max! + 5 }).settings.headingLimit.value).toBe(range.max)
    expect(read({ headingLimit: range.min! - 5 }).settings.headingLimit.value).toBe(range.min)
    expect(read({ headingLimit: 2.5 }).settings.headingLimit.value).toBe(range.default)
  })

  it('follow the person changing them', () => {
    const { context, settings } = read()

    context.setSetting('nestHeadings', false)
    context.setSetting('headingLimit', 3)

    expect([settings.isNesting.value, settings.headingLimit.value]).toStrictEqual([false, 3])
  })
})
