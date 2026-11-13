import { describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import { HEADING_LIMIT, NEST_HEADINGS, readNesting } from './nesting'
import { readLook } from './look'

const readLookFrom = (settings: Record<string, unknown>) =>
  readLook({
    settings: { get: (key: string) => settings[key], onChange: () => ({ dispose: vi.fn() }) },
  } as unknown as Pick<PluginContext, 'settings'>)

describe('the keys of the plex look settings', () => {
  it('reads every look setting from its camelCase key', () => {
    const look = readLookFrom({
      showPills: false,
      highlightLinks: true,
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
      references: 'incoming',
    })

    expect(look.showPills.value).toBe(false)
    expect(look.highlightLinks.value).toBe(true)
    expect(look.pillPaddingSides.value).toBe(4)
    expect(look.pillPaddingEnds.value).toBe(3.5)
    expect(look.focusSize.value).toBe(1.6)
    expect(look.animationLength.value).toBe(900)
    expect(look.lineThickness.value).toBe(3)
    expect(look.outlineThickness.value).toBe(1.5)
    expect(look.lineCurvature.value).toBe(0.2)
    expect(look.noteRoom.value).toBe(12)
    expect(look.namesOpacity.value).toBe(0.2)
    expect(look.focusedNamesOpacity.value).toBe(0.8)
    expect(look.references.value).toBe('incoming')
  })

  it('ignores the snake_case keys of the same settings', () => {
    const look = readLookFrom({
      show_pills: false,
      pill_padding_sides: 4,
      pill_padding_ends: 3.5,
      focus_size: 1.6,
      animation_length: 900,
      line_thickness: 3,
      outline_thickness: 1.5,
      line_curvature: 0.2,
      note_room: 12,
    })

    expect(look.showPills.value).toBe(true)
    expect(look.pillPaddingSides.value).toBe(10)
    expect(look.pillPaddingEnds.value).toBe(9.25)
    expect(look.focusSize.value).toBe(1.22)
    expect(look.animationLength.value).toBe(420)
    expect(look.lineThickness.value).toBe(2)
    expect(look.outlineThickness.value).toBe(1)
    expect(look.lineCurvature.value).toBe(0.55)
    expect(look.noteRoom.value).toBe(0)
  })
})

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
