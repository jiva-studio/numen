/** The names a bar plugin and the window share to find the bar slots and their entries in the page. */
import { describe, expect, it } from 'vitest'
import * as sdk from '../index'

describe('the bar slots', () => {
  it('are the buttons and the status of the title bar and the start and the end of the status bar, in that order', () => {
    expect((sdk as Record<string, unknown>)['BAR_SLOTS']).toStrictEqual([
      'titleBar.commands',
      'titleBar.status',
      'statusBar.left',
      'statusBar.right',
    ])
  })

  it('are marked in the page by the slot attribute and the entry attribute', () => {
    expect((sdk as Record<string, unknown>)['SLOT_ATTRIBUTE']).toBe('data-numen-slot')
    expect((sdk as Record<string, unknown>)['SLOT_ENTRY_ATTRIBUTE']).toBe('data-numen-slot-entry')
  })
})
