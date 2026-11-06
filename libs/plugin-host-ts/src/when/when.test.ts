/** The conditions of when clauses against libs/protocol/when.json, the vectors the Go reader runs as well. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { WHEN_KEYS } from '@numen/plugin-sdk'
import { evaluateWhen, type WhenValues } from './evaluate'
import { listWhenKeys, parseWhen } from './parse'

interface Vectors {
  readonly keys: readonly { readonly name: string }[]
  readonly valid: readonly {
    readonly when: string
    readonly keys: readonly string[]
    readonly values: readonly { readonly context: Partial<WhenValues>; readonly value: boolean }[]
  }[]
  readonly invalid: readonly { readonly when: string; readonly about: string }[]
}

const VECTORS = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../../../protocol/when.json'), 'utf8'),
) as Vectors

const NONE: WhenValues = { isRoot: false, isFolder: false, isMultiple: false }

describe('a condition', () => {
  it('names the keys of when.json, as the SDK lists them', () => {
    expect(WHEN_KEYS).toStrictEqual(VECTORS.keys.map((one) => one.name))
  })

  it.each(VECTORS.valid)('$when is read into a tree of its keys', (one) => {
    const parsed = parseWhen(one.when)

    expect(parsed.ok).toBe(true)
    if (parsed.ok) expect(listWhenKeys(parsed.value)).toStrictEqual(one.keys)
  })

  it.each(VECTORS.valid)('$when has its value for the values of its keys', (one) => {
    const parsed = parseWhen(one.when)
    if (!parsed.ok) throw new Error(parsed.error)

    for (const { context, value } of one.values) {
      expect(evaluateWhen(parsed.value, { ...NONE, ...context }), JSON.stringify(context)).toBe(
        value,
      )
    }
  })

  it.each(VECTORS.invalid)('is not read where it is $about', (one) => {
    expect(parseWhen(one.when).ok).toBe(false)
  })

  it('names the key it does not know in its error', () => {
    const parsed = parseWhen('isFile && isFolder')

    expect(parsed.ok ? '' : parsed.error).toContain('"isFile"')
  })
})
