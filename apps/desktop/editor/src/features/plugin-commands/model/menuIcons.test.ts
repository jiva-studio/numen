/** Every row the application and the shared commands place in a menu is drawn with an icon. */
import { describe, expect, it } from 'vitest'
import { SHARED_COMMANDS } from '@numen/plugin-loader'
import { findIcon } from '@/shared/model/command'
import { APP_SLOT_ENTRIES } from '../lib/appPlacements'

describe('the icons of the rows of a menu', () => {
  it.each(APP_SLOT_ENTRIES.flatMap((one) => (one.command ? [one.command] : [])))(
    'the application command %s has one',
    (command) => {
      expect(findIcon(command)).not.toBeNull()
    },
  )

  it.each(SHARED_COMMANDS.map((one) => one.id))('the shared command %s has one', (command) => {
    expect(findIcon(command)).not.toBeNull()
  })
})
