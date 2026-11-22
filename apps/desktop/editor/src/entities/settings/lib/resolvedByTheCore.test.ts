/**
 * The rules of the options of a setting and of its condition are applied once, by the core. The
 * window draws what the core resolved and holds no copy of the rules.
 */
import { describe, expect, it } from 'vitest'
import * as settings from '@/entities/settings'

describe('the rules of options and conditions', () => {
  it('are not exported by the settings of the window', () => {
    for (const name of ['findSetting', 'formatValue', 'getOptions', 'getValue', 'isShown']) {
      expect(Object.keys(settings), name).not.toContain(name)
    }
  })
})
