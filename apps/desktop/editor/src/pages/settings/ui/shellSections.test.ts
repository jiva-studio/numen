/**
 * The settings page keeps no section of its own for the shell: one renderer draws the settings of
 * every owner.
 */
import { describe, expect, it } from 'vitest'

describe('the settings page', () => {
  it('has no section of its own for the window or for the names of notes', () => {
    const files = Object.keys(import.meta.glob('./**/*.vue'))

    expect(files.filter((one) => /WindowSettingsSection|NamingSettings/.test(one))).toStrictEqual(
      [],
    )
  })
})
