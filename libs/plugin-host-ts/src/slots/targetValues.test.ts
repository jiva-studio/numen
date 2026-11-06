/** The values of the keys of when clauses over the target of a slot. Every path here is invented. */
import { describe, expect, it } from 'vitest'
import type { CommandTarget } from '@numen/plugin-sdk'
import { getTargetWhenValues } from './targetValues'

const target = (path: string, isFolder: boolean, others: string[] = []): CommandTarget => ({
  path,
  title: path,
  isFolder,
  others,
})

describe('the values of a target', () => {
  it('say whether it is the root folder, a folder, and one of several', () => {
    expect(getTargetWhenValues(target('', true))).toStrictEqual({
      isRoot: true,
      isFolder: true,
      isMultiple: false,
    })
    expect(getTargetWhenValues(target('notes', true))).toStrictEqual({
      isRoot: false,
      isFolder: true,
      isMultiple: false,
    })
    expect(getTargetWhenValues(target('notes/Bees.md', false, ['notes/Wasps.md']))).toStrictEqual({
      isRoot: false,
      isFolder: false,
      isMultiple: true,
    })
  })

  it('are all false where the place has no target', () => {
    expect(getTargetWhenValues(undefined)).toStrictEqual({
      isRoot: false,
      isFolder: false,
      isMultiple: false,
    })
  })
})
