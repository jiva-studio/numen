/**
 * The rules of the options of a setting and of its condition are applied once, by the core. The
 * window draws what the core resolved and holds no copy of the rules.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import * as settings from '@/entities/settings'

const here = dirname(fileURLToPath(import.meta.url))
const src = join(here, '..', '..', '..')

const listSources = (dir: string): string[] =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) return listSources(path)
    return /\.(ts|vue)$/.test(entry.name) && !/\.(test|stories)\.ts$/.test(entry.name) ? [path] : []
  })

describe('the rules of options and conditions', () => {
  it('have no copy in the window', () => {
    expect(existsSync(join(here, 'linked.ts'))).toBe(false)
  })

  it('are not exported by the settings of the window', () => {
    for (const name of ['findSetting', 'formatValue', 'getOptions', 'getValue', 'isShown']) {
      expect(Object.keys(settings), name).not.toContain(name)
    }
  })

  it('are applied by no source of the window', () => {
    const readers = listSources(src).filter((file) => {
      const text = readFileSync(file, 'utf8')
      return (
        /\.optionsBy\b/.test(text) &&
        !/(entities\/settings\/api\/settings|testing\/settingsWire)\.ts$/.test(file)
      )
    })
    expect(readers).toStrictEqual([])
  })
})
