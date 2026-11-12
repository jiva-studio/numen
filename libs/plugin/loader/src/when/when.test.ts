/** The when conditions against libs/core/protocol/when.json, the examples the Go evaluator runs as well. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { WHEN_VARIABLES } from '@numen/plugin-sdk'
import { getTargetValues, type TargetNote } from './targetValues'
import { parseWhen, type WhenCondition, type WhenValues } from './when'

type WhenResult = ReturnType<WhenCondition['evaluatePartial']>

interface Vectors {
  readonly variables: readonly { readonly name: string; readonly type: string }[]
  readonly examples: readonly {
    readonly when: string
    readonly values: readonly { readonly context: WhenValues; readonly value: boolean }[]
  }[]
  readonly partial: readonly {
    readonly when: string
    readonly values: readonly {
      readonly context: WhenValues
      readonly value: boolean | 'unknown'
    }[]
  }[]
  readonly targets: {
    readonly contentTypes: Readonly<Record<string, string>>
    readonly cases: readonly {
      readonly about: string
      readonly target: { readonly path: string; readonly isFolder: boolean }
      readonly note?: TargetNote
      readonly values: Readonly<Record<string, unknown>>
    }[]
  }
  readonly invalid: readonly {
    readonly when: string
    readonly about: string
    readonly isTypeError?: boolean
  }[]
}

const VECTORS = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../../../../core/protocol/when.json'), 'utf8'),
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

  it.each(VECTORS.partial)('$when has its value where variables are left out', (example) => {
    const condition = read(example.when)

    for (const { context, value } of example.values) {
      expect(condition.evaluatePartial(context), JSON.stringify(context)).toBe(
        String(value) as WhenResult,
      )
    }
  })

  it.each(VECTORS.targets.cases)('derives the target values of $about', (one) => {
    expect(
      getTargetValues({ ...one.target, others: [] }, VECTORS.targets.contentTypes, one.note),
    ).toStrictEqual(one.values)
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

describe('a when condition evaluated over what is known', () => {
  it.each([
    [
      "target.noteType == 'link' && version.kind == 'captions'",
      { 'target.noteType': 'deck' },
      'false',
    ],
    [
      "target.noteType == 'link' && version.kind == 'captions'",
      { 'target.noteType': 'link' },
      'unknown',
    ],
    ['version.isLive || target.isFolder', { 'target.isFolder': true }, 'true'],
    ['version.isLive || target.isFolder', { 'target.isFolder': false }, 'unknown'],
    [
      "has(target.properties.media) || target.noteType == 'link'",
      { 'target.noteType': 'deck' },
      'unknown',
    ],
    ["target.contentType == 'application/pdf'", { 'target.contentType': 'audio/mpeg' }, 'false'],
    [
      "target.contentType == 'application/pdf'",
      { 'target.contentType': 'application/pdf' },
      'true',
    ],
    ['', {}, 'true'],
  ] as const)('%s over %o is %s', (text, values, want) => {
    expect(read(text).evaluatePartial(values)).toBe(want)
  })

  it('is the value of the full evaluation where every variable it reads is given', () => {
    const condition = read("target.isNote && target.noteType == 'deck'")
    const values = { 'target.isNote': true, 'target.noteType': 'deck' }

    expect(condition.evaluatePartial(values)).toBe('true')
    expect(condition.evaluate(values)).toBe(true)
  })
})
