import { describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import { HEADING_LIMIT, NEST_HEADINGS, readNesting } from './nesting'

describe('the keys of the plex graph settings', () => {
  it('names the two settings in camelCase', () => {
    expect(NEST_HEADINGS).toBe('nestHeadings')
    expect(HEADING_LIMIT).toBe('headingLimit')
  })

  it('reads the nesting and the limit from the camelCase keys', () => {
    const settings: Record<string, unknown> = { nestHeadings: false, headingLimit: 3 }
    const nesting = readNesting({
      settings: { get: (key: string) => settings[key], onChange: () => ({ dispose: vi.fn() }) },
    } as unknown as Pick<PluginContext, 'settings'>)

    expect(nesting.isNesting.value).toBe(false)
    expect(nesting.headings.value).toBe(3)
  })

  it('ignores the snake_case keys of the same settings', () => {
    const settings: Record<string, unknown> = { nest_headings: false, heading_limit: 3 }
    const nesting = readNesting({
      settings: { get: (key: string) => settings[key], onChange: () => ({ dispose: vi.fn() }) },
    } as unknown as Pick<PluginContext, 'settings'>)

    expect(nesting.isNesting.value).toBe(true)
    expect(nesting.headings.value).toBe(6)
  })
})
