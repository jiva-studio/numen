/**
 * A setting typed by hand, read.
 *
 * The awkward ones are the ones a comment or a comma is written inside a string
 * of: what stands in quotes is what a person typed, and nothing there is a
 * comment or a comma between members.
 */
import { describe, expect, it } from 'vitest'
import { plain, read, write, Unreadable } from './json5'

describe('what JSON already holds', () => {
  it('is read as it stands', () => {
    expect(read('{"model": "opus", "max_steps": 30}')).toStrictEqual({
      model: 'opus',
      max_steps: 30,
    })
  })

  it('is read through every kind of value', () => {
    expect(read('{"a": [1, 2.5, -3], "b": true, "c": null, "d": {}}')).toStrictEqual({
      a: [1, 2.5, -3],
      b: true,
      c: null,
      d: {},
    })
  })
})

describe('a comment', () => {
  it('is taken out to the end of the line', () => {
    expect(read('{\n  // which model answers\n  "model": "opus"\n}')).toStrictEqual({
      model: 'opus',
    })
  })

  it('is taken out where it runs over several lines', () => {
    expect(read('{/* what this\n is for */ "model": "opus"}')).toStrictEqual({ model: 'opus' })
  })

  it('is not a comment inside a string', () => {
    expect(read('{"url": "https://numen.md/a//b"}')).toStrictEqual({
      url: 'https://numen.md/a//b',
    })
  })

  it('nothing closes is refused', () => {
    expect(() => read('{"a": 1} /* on and on')).toThrow(Unreadable)
  })
})

describe('a comma after the last member', () => {
  it('is taken out of an object', () => {
    expect(read('{"model": "opus",}')).toStrictEqual({ model: 'opus' })
  })

  it('is taken out of a list', () => {
    expect(read('["claude", "--verbose",\n]')).toStrictEqual(['claude', '--verbose'])
  })

  it('is not a comma inside a string', () => {
    expect(read('{"said": "one, two,"}')).toStrictEqual({ said: 'one, two,' })
  })
})

describe('a key', () => {
  it('is read without its quotes', () => {
    expect(read('{model: "opus"}')).toStrictEqual({ model: 'opus' })
  })

  it('is read with them', () => {
    expect(read('{"model": "opus"}')).toStrictEqual({ model: 'opus' })
  })

  it('is not a word standing where a value stands', () => {
    expect(read('{"on": true, "off": false, "none": null}')).toStrictEqual({
      on: true,
      off: false,
      none: null,
    })
  })
})

describe('a string', () => {
  it('is read in single quotes as it is in double', () => {
    expect(read("{'model': 'opus'}")).toStrictEqual({ model: 'opus' })
  })

  it('holds the quotes of the other kind as they were typed', () => {
    expect(read(`{'said': 'he said "no"'}`)).toStrictEqual({ said: 'he said "no"' })
  })

  it('reads an escape as what it stands for', () => {
    expect(read(`{"said": "one\\ttwo\\u0041"}`)).toStrictEqual({ said: 'one\ttwoA' })
  })

  it('is one line where a line is broken with a backslash', () => {
    expect(read("{'said': 'one\\\ntwo'}")).toStrictEqual({ said: 'onetwo' })
  })

  it('nothing closes is refused', () => {
    expect(() => read('{"said": "on and on')).toThrow(Unreadable)
  })
})

describe('what JSON5 holds and JSON does not', () => {
  it('is refused where a number is written the way only JSON5 writes it', () => {
    expect(() => read('{"floor": .5}')).toThrow()
    expect(() => read('{"floor": Infinity}')).toThrow()
  })
})

describe('what is written back', () => {
  it('is JSON, laid out to be read', () => {
    expect(write({ model: 'opus' })).toBe('{\n  "model": "opus"\n}')
  })

  it('is read back as what was written', () => {
    expect(read(write({ a: [1, 'two'], b: null }))).toStrictEqual({ a: [1, 'two'], b: null })
  })
})

// The text a person typed is turned into JSON before it is read, so what that
// leaves is worth asking about on its own.
describe('the text as JSON', () => {
  it('keeps the strings whole', () => {
    expect(plain(`{a: 'one', /* two */ b: "three",}`)).toBe('{"a": "one",  "b": "three"}')
  })
})
