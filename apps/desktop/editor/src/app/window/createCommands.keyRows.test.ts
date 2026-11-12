/** A key the application holds is drawn on a row of the palette that carries it out. */
import { describe, expect, it } from 'vitest'
import { APP_KEYBINDINGS } from '@numen/plugin-loader'
import { createRowKeys } from '@/shared/model/command'
import { createAppCommands } from '@/testing/appCommands'

const APPLE = 'MacIntel'
const OPENS_A_PALETTE = ['app:vault.search', 'app:commands.open']
const appKeys = APP_KEYBINDINGS.filter((one) => !OPENS_A_PALETTE.includes(one.command))

describe('every key the application holds', () => {
  it('names a row the palette has, unless it opens a palette', () => {
    const recognized = new Set(createAppCommands(APPLE).map((one) => one.id))
    for (const one of appKeys) expect(recognized.has(one.command)).toBe(true)
  })

  it('is drawn on the row of the command it names', () => {
    const commands = createAppCommands(APPLE)
    for (const one of appKeys) {
      const row = commands.find((command) => command.id === one.command)
      expect(row?.keys).toEqual(createRowKeys(one.command, APPLE).keys)
    }
  })
})
