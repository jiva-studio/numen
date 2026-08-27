/**
 * The braces, read and filled. Plain functions over plain values: what comes
 * out is the same text on any machine on any day.
 */
import { describe, expect, it } from 'vitest'
import {
  braced,
  fill,
  insert,
  previewed,
  renamedIn,
  sampled,
  slotsIn,
  strayIn,
} from './fill'

const VALUES = [
  { field: 'Height', text: 'about 45"' },
  { field: 'Life span', text: 'about 20 years' },
]

describe('slotsIn', () => {
  it('finds every slot where it stands', () => {
    expect(slotsIn('a {{One}} b {{Two}}')).toEqual([
      { field: 'One', from: 2, to: 9 },
      { field: 'Two', from: 12, to: 19 },
    ])
  })

  it('drops the space around a name', () => {
    expect(slotsIn('{{  Life span  }}')[0]?.field).toBe('Life span')
  })

  it('finds nothing in text with no braces', () => {
    expect(slotsIn('plain words, and a { brace } that is not doubled')).toEqual([])
  })

  it('takes an empty pair of braces as a slot with no name', () => {
    expect(slotsIn('{{}}')).toEqual([{ field: '', from: 0, to: 4 }])
  })
})

describe('fill', () => {
  it('stands a value in every slot that names a field', () => {
    expect(fill('**Height:** {{Height}}', VALUES)).toBe('**Height:** about 45"')
  })

  it('stands the naming field like any other', () => {
    expect(fill('{{Name}} is tall', [{ field: 'Name', text: 'Llama' }])).toBe('Llama is tall')
  })

  it('leaves a slot nothing was handed for empty', () => {
    expect(fill('[{{Weight}}]', VALUES)).toBe('[]')
  })

  it('keeps the markup around the slots', () => {
    expect(fill('- {{Height}}\n- {{Life span}}\n', VALUES)).toBe(
      '- about 45"\n- about 20 years\n',
    )
  })

  it('stands a value that is itself markdown', () => {
    const values = [{ field: 'Picture', text: '![[llama.jpg]]' }]
    expect(fill('{{Picture}}', values)).toBe('![[llama.jpg]]')
  })

  it('does not read the braces a value stands in the text', () => {
    const values = [{ field: 'One', text: '{{Two}}' }]
    expect(fill('{{One}}', [...values, { field: 'Two', text: 'caught' }])).toBe('{{Two}}')
  })

  it('fills text that is not Latin', () => {
    const values = [
      { field: 'Перевод', text: 'слово' },
      { field: 'Слово', text: 'Карточка' },
    ]
    expect(fill('{{Перевод}} — {{Слово}}', values)).toBe('слово — Карточка')
  })
})

describe('previewed', () => {
  it('stands a value in every slot the fields name', () => {
    expect(previewed('{{Height}}', VALUES, ['Height'])).toBe('about 45"')
  })

  it('leaves a slot the fields do not name in its braces, marked where it stands', () => {
    expect(previewed('a {{Colour}} b', VALUES, ['Height'])).toBe('a <mark>{{Colour}}</mark> b')
  })

  it('marks every stray slot where each of them stands', () => {
    expect(previewed('{{A}}{{Height}}{{B}}', VALUES, ['Height'])).toBe(
      '<mark>{{A}}</mark>about 45"<mark>{{B}}</mark>',
    )
  })

  it('stands a marked slot as text, so its braces draw no tags', () => {
    expect(previewed('{{<img src=x>}}', [], [])).toBe(
      '<mark>{{&lt;img src=x&gt;}}</mark>',
    )
  })

  it('leaves a slot with no name empty, and does not mark it', () => {
    expect(previewed('[{{}}]', [], ['Height'])).toBe('[]')
  })
})

describe('strayIn', () => {
  it('says the slots the fields do not name', () => {
    expect(strayIn('{{Height}} {{Colour}}', ['Height'])).toEqual(['Colour'])
  })

  it('says each of them once', () => {
    expect(strayIn('{{Colour}} {{Colour}}', [])).toEqual(['Colour'])
  })

  it('says nothing of the naming field, which is named like any other', () => {
    expect(strayIn('{{Name}}', ['Name'])).toEqual([])
  })

  it('says nothing of a slot with no name', () => {
    expect(strayIn('{{}}', [])).toEqual([])
  })
})

describe('insert', () => {
  it('writes the field where the caret stands', () => {
    expect(insert('ab', 1, 'X')).toEqual({ text: 'a{{X}}b', caret: 6 })
  })

  it('lands the caret past the closing brace', () => {
    const done = insert('one two', 3, 'Life span')
    expect(done.text).toBe('one{{Life span}} two')
    expect(done.text.slice(0, done.caret)).toBe('one{{Life span}}')
  })

  it('takes a caret past the end to the end, the caret with it', () => {
    expect(insert('ab', 99, 'X')).toEqual({ text: 'ab{{X}}', caret: 7 })
  })

  it('takes a caret before the start to the start, the caret with it', () => {
    expect(insert('ab', -3, 'X')).toEqual({ text: '{{X}}ab', caret: 5 })
  })
})

describe('renamedIn', () => {
  it('rewrites every slot naming the field', () => {
    expect(renamedIn('{{A}} {{B}} {{A}}', 'A', 'C')).toBe('{{C}} {{B}} {{C}}')
  })

  it('leaves the field’s name in the prose alone', () => {
    expect(renamedIn('A is {{A}}', 'A', 'C')).toBe('A is {{C}}')
  })
})

describe('sampled', () => {
  it('stands each field under its own name', () => {
    expect(sampled(['Height', 'Weight'])).toEqual([
      { field: 'Height', text: 'Height' },
      { field: 'Weight', text: 'Weight' },
    ])
  })
})

describe('braced', () => {
  it('is what a slot is written as', () => {
    expect(slotsIn(braced('Life span'))[0]?.field).toBe('Life span')
  })
})
