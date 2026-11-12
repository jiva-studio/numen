import { describe, expect, it } from 'vitest'
import { getListedCommandIds, isCommandListed } from './listedCommands'

const createHost = (ids: readonly string[]) => ({
  commands: { getPaletteCommands: () => ids.map((id) => ({ id })) },
})

describe('the commands the plugins list in the palette', () => {
  it('have the ids the host reports, in its order', () => {
    expect(getListedCommandIds(createHost(['sample.b', 'sample.a']))).toStrictEqual([
      'sample.b',
      'sample.a',
    ])
  })

  it('are none while the host reports none', () => {
    expect(getListedCommandIds(createHost([]))).toStrictEqual([])
  })

  it('include a command of an id the host reports', () => {
    expect(isCommandListed(createHost(['sample.a']), 'sample.a')).toBe(true)
  })

  it('leave out a command of an id the host does not report', () => {
    expect(isCommandListed(createHost(['sample.a']), 'sample.b')).toBe(false)
  })
})
