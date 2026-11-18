/** The plex draws the lines between notes and the views plugins place at its nodes; the names of related notes are not its own. */
import { describe, expect, it } from 'vitest'
import * as plex from './index'

const NAMES_MODULE =
  /(^|\/)(PlexNodeNames|names|namesOpacity|namesDefaults|useNamesRooms)(\.[a-z]+)*\.(ts|vue)$/

describe('the plex', () => {
  it('holds no module that places or draws the names of related notes', () => {
    const modules = Object.keys(
      import.meta.glob([
        '../**/*.ts',
        '../**/*.vue',
        '!../**/*.test.ts',
        '!../**/*.stories.ts',
        '!../fixtures/**',
      ]),
    )

    expect(modules.filter((one) => NAMES_MODULE.test(one))).toStrictEqual([])
  })

  it('exports no default opacity of names', () => {
    expect(Object.keys(plex).filter((key) => /names/i.test(key))).toStrictEqual([])
  })
})
