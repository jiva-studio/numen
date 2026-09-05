/**
 * What a model is called, and what the list of them offers.
 *
 * The list is built from the file first: what a person has configured is on it
 * and is the one in force, whatever the presets say. Every value here is
 * invented.
 */
import { describe, expect, it } from 'vitest'
import type { Model } from '../../core'
import { choicesFor, nameOf } from './models'

const WORDS = {
  byDefault: 'the default',
  owned: 'Yours',
  present: 'on this machine',
  notFetched: 'not fetched yet',
}

const model = (one: Partial<Model> & { name: string }): Model => ({
  namedAt: ['indexing', 'embedding', 'model', 'name'],
  title: '',
  shelf: '',
  byDefault: false,
  writes: [],
  presence: 'nothing to fetch',
  ...one,
})

describe('what a model is called', () => {
  it('is the value itself where it is one word', () => {
    expect(nameOf('opus')).toBe('opus')
  })

  it('is the last segment of a repository', () => {
    expect(nameOf('somewhere/tiny-e5-small')).toBe('tiny-e5-small')
  })

  it('is the last segment of an address, without what is asked of it', () => {
    expect(nameOf('https://models.example/held/v1/rec/Tiny_rec.onnx?download=1')).toBe(
      'Tiny_rec.onnx',
    )
  })

  it('is the last segment of a path, whatever it ends with', () => {
    expect(nameOf('/held/models/Tiny_rec.onnx')).toBe('Tiny_rec.onnx')
    expect(nameOf('/held/models/')).toBe('models')
  })

  it('is nothing where the value is nothing', () => {
    expect(nameOf('')).toBe('')
  })
})

describe('the choices a setting offers', () => {
  const PRESET = model({
    name: 'https://models.example/held/Tiny_rec.onnx',
    title: 'Tiny, small',
    byDefault: true,
  })

  it('name a preset by its own words and address it underneath', () => {
    const [one] = choicesFor([PRESET], PRESET.name, WORDS)
    expect(one?.text).toBe('Tiny, small — the default')
    expect(one?.detail).toBe(PRESET.name)
  })

  it('name a model by its last segment where the build gives it an address for a name', () => {
    const own = 'https://models.example/held/v3/rec/eslav_rec_mobile.onnx'
    // What the vault answers with for a value standing in the settings: the
    // address in the place a name would be.
    const [one] = choicesFor([model({ name: own, title: own })], own, WORDS)

    expect(one?.text).toBe('eslav_rec_mobile.onnx')
    expect(one?.detail).toContain(own)
  })

  it('leave a preset named by one word without an address under it', () => {
    const [one] = choicesFor([model({ name: 'opus', title: 'Opus' })], 'opus', WORDS)
    expect(one?.text).toBe('Opus')
    expect(one?.detail).toBeUndefined()
  })

  it('put a value the presets do not name first, as the person’s own', () => {
    const own = 'https://models.example/mine/Other_rec.onnx'
    const offered = choicesFor([PRESET], own, WORDS)

    expect(offered.map((one) => one.id)).toStrictEqual([own, PRESET.name])
    expect(offered[0]).toStrictEqual({
      id: own,
      text: 'Other_rec.onnx',
      detail: own,
      group: 'Yours',
    })
  })

  it('say nothing about a value being missing, wrong or not found', () => {
    const said = choicesFor([PRESET], 'somewhere/mine', WORDS)
      .map((one) => `${one.text} ${one.detail ?? ''}`)
      .join(' ')
    expect(said).not.toMatch(/not found|missing|unknown/i)
  })

  it('offer the presets alone where the file names one of them', () => {
    expect(choicesFor([PRESET], PRESET.name, WORDS)).toHaveLength(1)
  })

  it('offer the presets alone where the file names nothing', () => {
    expect(choicesFor([PRESET], '', WORDS)).toHaveLength(1)
  })

  it('say what a model’s files are on this machine, where that means anything', () => {
    const here = model({ name: 'somewhere/tiny', title: 'Tiny', presence: 'present' })
    const coming = model({ name: 'somewhere/large', title: 'Large', presence: 'not fetched' })
    const reached = model({ name: 'opus', title: 'Opus', presence: 'nothing to fetch' })

    const offered = choicesFor([here, coming, reached], 'opus', WORDS)
    expect(offered.map((one) => one.detail)).toStrictEqual([
      'on this machine · somewhere/tiny',
      'not fetched yet · somewhere/large',
      undefined,
    ])
  })

  it('say nothing about presence for a model reached over the network', () => {
    const [one] = choicesFor(
      [model({ name: 'https://held.example/one', title: 'One', presence: 'nothing to fetch' })],
      '',
      WORDS,
    )
    expect(one?.detail).toBe('https://held.example/one')
  })

  it('stand each preset on the shelf it names', () => {
    const offered = choicesFor(
      [
        model({ name: 'opus', title: 'Opus', shelf: 'By how large it is' }),
        model({ name: 'held/claude-opus-5', title: 'claude-opus-5', shelf: 'In full' }),
      ],
      'opus',
      WORDS,
    )
    expect(offered.map((one) => one.group)).toStrictEqual(['By how large it is', 'In full'])
  })
})
