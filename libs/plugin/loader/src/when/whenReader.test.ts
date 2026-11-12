/** The conditions of a window, each read once, and the values they are evaluated over. */
import { describe, expect, it } from 'vitest'
import { createWhenReader } from './whenReader'
import { getWhenValues } from './whenValues'

describe('a reader of conditions', () => {
  it('reads a text once and gives the same condition again', () => {
    const reader = createWhenReader()

    expect(reader.read('target.isNote')).toBe(reader.read('target.isNote'))
    expect(reader.getCondition('target.isNote')?.names).toStrictEqual(['target.isNote'])
  })

  it('keeps nothing between two readers', () => {
    expect(createWhenReader().read('target.isNote')).not.toBe(
      createWhenReader().read('target.isNote'),
    )
  })

  it('gives no condition for a text it cannot read, and the always-true one for none', () => {
    const reader = createWhenReader()

    expect(reader.getCondition('target.isRemote')).toBeUndefined()
    expect(reader.getCondition(undefined)?.evaluate({})).toBe(true)
  })
})

describe('the values of an evaluation', () => {
  it('are the window variables, the target derived from its path and the settings', () => {
    expect(
      getWhenValues({
        window: { 'window.isVaultOpen': true },
        target: { path: 'talks/River.md', isFolder: false, others: [] },
        note: { type: 'link' },
        contentTypes: { md: 'text/markdown' },
        config: { editor: { lineNumbers: true } },
      }),
    ).toStrictEqual({
      'window.isVaultOpen': true,
      'target.isRoot': false,
      'target.isFolder': false,
      'target.isMultiple': false,
      'target.isNote': true,
      'target.isFile': false,
      'target.contentType': 'text/markdown',
      'target.noteType': 'link',
      'target.properties': {},
      config: { editor: { lineNumbers: true } },
    })
  })
})
