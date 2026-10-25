/**
 * What a model is called, and what the list of them offers.
 *
 * The list is built from the file first: what a person has configured is on it
 * and is the one in force, whatever the presets say. Every value here is
 * invented.
 */
import { describe, expect, it } from 'vitest'
import type { Model } from '@/entities/settings'
import { createChoices, createModelChoices, getLastPathSegment } from './models'

const WORDS = {
  byDefault: 'the default',
  owned: 'Yours',
  present: 'on this machine',
  notFetched: 'not fetched yet',
}

const model = (one: Partial<Model> & { name: string }): Model => ({
  namedAt: ['indexing', 'embedding', 'model', 'name'],
  title: '',
  group: '',
  isDefault: false,
  writes: [],
  presence: 'nothing to fetch',
  ...one,
})

describe('what a model is called', () => {
  it('is the value itself where it is one word', () => {
    expect(getLastPathSegment('opus')).toBe('opus')
  })

  it('is the last segment of a repository', () => {
    expect(getLastPathSegment('somewhere/tiny-e5-small')).toBe('tiny-e5-small')
  })

  it('is the last segment of an address, without what is requested of it', () => {
    expect(getLastPathSegment('https://models.example/held/v1/rec/Tiny_rec.onnx?download=1')).toBe(
      'Tiny_rec.onnx',
    )
  })

  it('is the last segment of a path, whatever it ends with', () => {
    expect(getLastPathSegment('/held/models/Tiny_rec.onnx')).toBe('Tiny_rec.onnx')
    expect(getLastPathSegment('/held/models/')).toBe('models')
  })

  it('is nothing where the value is nothing', () => {
    expect(getLastPathSegment('')).toBe('')
  })
})

describe('the choices a setting offers', () => {
  const PRESET = model({
    name: 'https://models.example/held/Tiny_rec.onnx',
    title: 'Tiny, small',
    isDefault: true,
  })

  it('name a preset by its own words and address it underneath', () => {
    const [one] = createChoices([PRESET], PRESET.name, WORDS)
    expect(one?.text).toBe('Tiny, small — the default')
    expect(one?.detail).toBe(PRESET.name)
  })

  it('name a model by its last segment where the build gives it an address for a name', () => {
    const own = 'https://models.example/held/v3/rec/eslav_rec_mobile.onnx'
    // address in the place a name would be.
    const [one] = createChoices([model({ name: own, title: own })], own, WORDS)

    expect(one?.text).toBe('eslav_rec_mobile.onnx')
    expect(one?.detail).toContain(own)
  })

  it('leave a preset named by one word without an address under it', () => {
    const [one] = createChoices([model({ name: 'opus', title: 'Opus' })], 'opus', WORDS)
    expect(one?.text).toBe('Opus')
    expect(one?.detail).toBeUndefined()
  })

  it('put a value the presets do not name first, as the person’s own', () => {
    const own = 'https://models.example/mine/Other_rec.onnx'
    const proposed = createChoices([PRESET], own, WORDS)

    expect(proposed.map((one) => one.id)).toStrictEqual([own, PRESET.name])
    expect(proposed[0]).toStrictEqual({
      id: own,
      text: 'Other_rec.onnx',
      detail: own,
      group: 'Yours',
    })
  })

  it('say nothing about a value being missing, wrong or not found', () => {
    const saidText = createChoices([PRESET], 'somewhere/mine', WORDS)
      .map((one) => `${one.text} ${one.detail ?? ''}`)
      .join(' ')
    expect(saidText).not.toMatch(/not found|missing|unknown/i)
  })

  it('offer the presets alone where the file names one of them', () => {
    expect(createChoices([PRESET], PRESET.name, WORDS)).toHaveLength(1)
  })

  it('offer the presets alone where the file is empty', () => {
    expect(createChoices([PRESET], '', WORDS)).toHaveLength(1)
  })

  it('say what a model’s files are on this machine, where that means anything', () => {
    const here = model({ name: 'somewhere/tiny', title: 'Tiny', presence: 'present' })
    const coming = model({ name: 'somewhere/large', title: 'Large', presence: 'not fetched' })
    const reached = model({ name: 'opus', title: 'Opus', presence: 'nothing to fetch' })

    const proposed = createChoices([here, coming, reached], 'opus', WORDS)
    expect(proposed.map((one) => one.detail)).toStrictEqual([
      'on this machine · somewhere/tiny',
      'not fetched yet · somewhere/large',
      undefined,
    ])
  })

  it('say nothing about presence for a model reached over the network', () => {
    const [one] = createChoices(
      [model({ name: 'https://held.example/one', title: 'One', presence: 'nothing to fetch' })],
      '',
      WORDS,
    )
    expect(one?.detail).toBe('https://held.example/one')
  })

  it('stand each preset on the group it names', () => {
    const proposed = createChoices(
      [
        model({ name: 'opus', title: 'Opus', group: 'By how large it is' }),
        model({ name: 'held/claude-opus-5', title: 'claude-opus-5', group: 'In full' }),
      ],
      'opus',
      WORDS,
    )
    expect(proposed.map((one) => one.group)).toStrictEqual(['By how large it is', 'In full'])
  })
})

describe('the choices of a field', () => {
  it('are the models offered, with the value the file holds standing in force', () => {
    const fields = {
      getModels: () => [model({ name: 'opus', title: 'Opus' })],
      getString: () => 'mine',
    }
    const proposed = createModelChoices(fields, ['agent', 'model'], WORDS)
    expect(proposed.map((one) => one.id)).toStrictEqual(['mine', 'opus'])
  })
})
