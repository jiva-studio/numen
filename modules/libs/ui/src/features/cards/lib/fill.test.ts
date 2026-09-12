/**
 * The braces, read and filled. Plain functions over plain values: what comes
 * out is the same text on any machine on any day.
 */
import { describe, expect, it } from 'vitest'
import {
  braceField,
  insert,
  renderPreview,
  renameField,
  sampleValues,
  slotsIn,
  strayIn,
} from './fill'

/**
 * A card face is shown on two surfaces and each lays it out for itself: this
 * library for the preview beside the stencil being written, and the core's
 * flashcards/format for the window a card is reviewed in. The table both are
 * held to is one file, and neither owns it.
 */
import corpus from '../../../../../protocol/testdata/faces.json'

const VALUES = [
  { field: 'Height', text: 'about 45"' },
  { field: 'Life span', text: 'about 20 years' },
]

/**
 * A card face means the same thing on every surface it is shown on.
 *
 * Laying a face out is the step each surface takes for itself; drawing what
 * comes out is one implementation both of them reach, so what is compared is
 * the text the slots have been filled in, and text that agrees is drawn alike.
 *
 * Every slot of the table names a field the stencil declares. A slot naming
 * none is where the two surfaces part on purpose — the review window lays it
 * out as nothing and the preview marks it where it stands, because the stencil
 * is what the preview is there to settle — and it is held to below instead.
 */
describe('a card face', () => {
  it('means the same on every surface it is shown on', () => {
    if (corpus.invariant === '') throw new Error('the table names no invariant')
    if (corpus.faces.length === 0 || corpus.count === 0 || corpus.changed === 0) {
      throw new Error(
        `the table holds ${corpus.faces.length} faces and declares ${corpus.count}, ` +
          `${corpus.changed} of which the slots change`,
      )
    }

    const named = new Set<string>()
    let laid = 0
    let changed = 0
    for (const { name, construct, face, fields, values, laid: want } of corpus.faces) {
      if (name === '' || construct === '') {
        throw new Error(`a face of the table is called "${name}" and pins "${construct}"`)
      }
      if (named.has(name)) throw new Error(`two faces of the table are called "${name}"`)
      named.add(name)

      laid += 1
      if (want !== face) changed += 1

      const said = { name, construct, laid: want }
      expect({ name, construct, laid: renderPreview(face, values, fields) }).toStrictEqual(said)
    }

    expect(laid).toBe(corpus.count)
    expect(changed).toBe(corpus.changed)
  })
})

describe('slotsIn', () => {
  it('finds every slot where it stands', () => {
    expect(slotsIn('a {{One}} b {{Two}}')).toEqual([
      { field: 'One', from: 2, to: 9 },
      { field: 'Two', from: 12, to: 19 },
    ])
  })

  it('reads the name between the braces as it is written, space and all', () => {
    expect(slotsIn('{{  Life span  }}')[0]?.field).toBe('  Life span  ')
  })

  it('finds nothing in text with no braces', () => {
    expect(slotsIn('plain words, and a { brace } that is not doubled')).toEqual([])
  })

  it('takes an empty pair of braces as a slot with no name', () => {
    expect(slotsIn('{{}}')).toEqual([{ field: '', from: 0, to: 4 }])
  })
})

describe('renderPreview', () => {
  it('stands a value in every slot the fields name', () => {
    expect(renderPreview('{{Height}}', VALUES, ['Height'])).toBe('about 45"')
  })

  it('stands the naming field like any other', () => {
    const values = [{ field: 'Name', text: 'Llama' }]
    expect(renderPreview('{{Name}} is tall', values, ['Name'])).toBe('Llama is tall')
  })

  it('leaves a slot nothing was handed for empty', () => {
    expect(renderPreview('[{{Weight}}]', VALUES, ['Weight'])).toBe('[]')
  })

  it('keeps the markup around the slots', () => {
    const fields = ['Height', 'Life span']
    expect(renderPreview('<li>{{Height}}</li>\n<li>{{Life span}}</li>\n', VALUES, fields)).toBe(
      '<li>about 45"</li>\n<li>about 20 years</li>\n',
    )
  })

  it('stands a value that is itself markup', () => {
    const values = [{ field: 'Picture', text: '<img src="llama.jpg" alt="a llama">' }]
    expect(renderPreview('{{Picture}}', values, ['Picture'])).toBe(
      '<img src="llama.jpg" alt="a llama">',
    )
  })

  it('does not read the braces a value stands in the text', () => {
    const values = [
      { field: 'One', text: '{{Two}}' },
      { field: 'Two', text: 'caught' },
    ]
    expect(renderPreview('{{One}}', values, ['One', 'Two'])).toBe('{{Two}}')
  })

  it('stands text that is not Latin', () => {
    const values = [
      { field: 'Перевод', text: 'compost' },
      { field: 'Слово', text: 'Компост' },
    ]
    expect(renderPreview('{{Перевод}} — {{Слово}}', values, ['Перевод', 'Слово'])).toBe(
      'compost — Компост',
    )
  })

  it('leaves a slot the fields do not name in its braces, marked where it stands', () => {
    expect(renderPreview('a {{Colour}} b', VALUES, ['Height'])).toBe('a <mark>{{Colour}}</mark> b')
  })

  it('marks every stray slot where each of them stands', () => {
    expect(renderPreview('{{A}}{{Height}}{{B}}', VALUES, ['Height'])).toBe(
      '<mark>{{A}}</mark>about 45"<mark>{{B}}</mark>',
    )
  })

  it('stands a marked slot as text, so its braces draw no tags', () => {
    expect(renderPreview('{{<img src=x>}}', [], [])).toBe(
      '<mark>{{&lt;img src=x&gt;}}</mark>',
    )
  })

  it('marks a slot whose name is written with space around it', () => {
    expect(renderPreview('{{ Height }}', VALUES, ['Height'])).toBe('<mark>{{ Height }}</mark>')
  })

  it('marks a slot with no name, which no field is called', () => {
    expect(renderPreview('[{{}}]', [], ['Height'])).toBe('[<mark>{{}}</mark>]')
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

  it('says a slot with no name, which no field is called', () => {
    expect(strayIn('{{}}', [])).toEqual([''])
  })

  it('says a slot whose name is written with space around it', () => {
    expect(strayIn('{{ Height }}', ['Height'])).toEqual([' Height '])
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

describe('renameField', () => {
  it('rewrites every slot naming the field', () => {
    expect(renameField('{{A}} {{B}} {{A}}', 'A', 'C')).toBe('{{C}} {{B}} {{C}}')
  })

  it('leaves the field’s name in the prose alone', () => {
    expect(renameField('A is {{A}}', 'A', 'C')).toBe('A is {{C}}')
  })

  it('rewrites no slot whose name is written with space around it', () => {
    expect(renameField('{{ A }}', 'A', 'C')).toBe('{{ A }}')
  })
})

describe('sampleValues', () => {
  it('stands each field under its own name', () => {
    expect(sampleValues(['Height', 'Weight'])).toEqual([
      { field: 'Height', text: 'Height' },
      { field: 'Weight', text: 'Weight' },
    ])
  })
})

describe('braceField', () => {
  it('is what a slot is written as', () => {
    expect(slotsIn(braceField('Life span'))[0]?.field).toBe('Life span')
  })
})
