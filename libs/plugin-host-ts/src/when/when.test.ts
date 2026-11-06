/** The when conditions against libs/protocol/when.json, the examples the Go evaluator runs as well. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { WHEN_VARIABLES } from '@numen/plugin-sdk'
import { parseWhen, type WhenValues } from './when'

interface Vectors {
  readonly variables: readonly { readonly name: string; readonly type: string }[]
  readonly examples: readonly {
    readonly when: string
    readonly values: readonly { readonly context: WhenValues; readonly value: boolean }[]
  }[]
  readonly invalid: readonly {
    readonly when: string
    readonly about: string
    readonly isTypeError?: boolean
  }[]
}

const VECTORS = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../../../protocol/when.json'), 'utf8'),
) as Vectors

const read = (text: string) => {
  const parsed = parseWhen(text)
  if (!parsed.ok) throw new Error(parsed.error)
  return parsed.value
}

describe('a when condition', () => {
  it('may name the variables of when.json, as the SDK lists them', () => {
    expect(WHEN_VARIABLES).toStrictEqual(
      VECTORS.variables.map(({ name, type }) => ({ name, type })),
    )
  })

  it.each(VECTORS.examples)('$when has its value in each context', (example) => {
    const condition = read(example.when)

    for (const { context, value } of example.values) {
      expect(condition.evaluate(context), JSON.stringify(context)).toBe(value)
    }
  })

  it.each(VECTORS.invalid.filter((one) => !one.isTypeError))(
    'is not read where it is $about',
    (one) => {
      expect(parseWhen(one.when).ok).toBe(false)
    },
  )

  it('holds always where it is empty, and reads no variable', () => {
    const condition = read('')

    expect(condition.evaluate({})).toBe(true)
    expect(condition.names).toStrictEqual([])
  })

  it('names the variables it reads, the settings as config', () => {
    const condition = read(
      'target.isNote && !window.isTabOpen || config.editor.lineNumbers == true',
    )

    expect(condition.names).toStrictEqual(['config', 'target.isNote', 'window.isTabOpen'])
  })

  it('takes a variable the context does not give as false or empty, and a missing setting as false', () => {
    expect(read('!target.isNote').evaluate({})).toBe(true)
    expect(read("target.contentType == ''").evaluate({})).toBe(true)
    expect(read('config.editor.lineNumbers == true').evaluate({ config: {} })).toBe(false)
  })

  it('names the variable it does not know in its error', () => {
    const parsed = parseWhen('target.isRemote && target.isNote')

    expect(parsed.ok ? '' : parsed.error).toContain('target.isRemote')
  })
})
