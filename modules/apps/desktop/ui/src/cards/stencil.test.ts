/**
 * What a stencil tab holds, asked without a screen: what it reads, what it
 * writes back, and where what is wrong with it stands.
 */
import { describe, expect, it } from 'vitest'
import type { Cards, Faced, Problem, Refused } from '../core'
import { windowing } from '../windowing'
import { STENCIL } from '../workspace'
import { stencilling, type Held } from './stencil'
import { WORDS as words } from './words'

/** A moment for whatever the tab asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

const FACES: readonly Faced[] = [
  { name: 'Recognise', front: '{{title}}', back: '**Height:** {{Height}}' },
]

/** A vault holding one stencil, writing down every write it was asked for. */
const vault = (
  answers: { refusal?: Refused; problems?: readonly Problem[]; changed?: boolean } = {},
) => {
  const written: string[] = []
  let fields: readonly string[] = ['Height', 'Life span']
  let faces: readonly Faced[] = FACES

  const core: Cards = {
    stencils: async () => ({ stencils: [], held: 0 }),
    readDeck: async () => ({ deck: null, refusal: 'missing', at: '', bound: 0 }),
    writeDeck: async () => ({ refusal: null, changed: false, at: '', bound: 0 }),
    readStencil: async (path) => {
      if (answers.refusal) return { stencil: null, refusal: answers.refusal, at: '' }
      return {
        stencil: {
          path,
          title: 'Animal',
          fields,
          faces,
          problems: answers.problems ?? [],
        },
        refusal: null,
        at: 'read',
      }
    },
    writeStencil: async (path, wrote, drew) => {
      written.push(`${path} ${wrote.join(', ') || '—'} | ${drew.map((one) => one.back).join(' ')}`)
      if (answers.changed) return { refusal: null, changed: true, at: '' }
      fields = wrote
      faces = drew
      return { refusal: null, changed: false, at: 'written' }
    },
  }

  return { core, written }
}

/** A window with one stencil open on a file, and what that tab holds. */
const open = async (answers: Parameters<typeof vault>[0] = {}, path = 'Animal.md') => {
  const one = vault(answers)
  const held = windowing()
  const stencils = stencilling(one.core, held.host)
  held.declares([stencils.kind])
  const id = await held.opens(STENCIL, path)
  await settles()
  const tab = held.host.holds<Held>(STENCIL, id) as Held
  return { ...one, held, stencils, id, tab }
}

describe('a stencil opened', () => {
  it('draws the fields in the order a person is asked for them', async () => {
    const { tab } = await open()

    expect(tab.sheet().fields).toStrictEqual(['Height', 'Life span'])
  })

  it('gives every face an identity, which the file carries none of', async () => {
    const { tab } = await open()

    expect(tab.sheet().faces[0]?.id).toBeTruthy()
    expect(tab.sheet().faces[0]?.name).toBe('Recognise')
  })

  it('is called what the file is called', async () => {
    const { stencils, tab } = await open()

    expect(stencils.kind.called(tab)).toBe('Animal')
  })
})

describe('a field renamed in a stencil', () => {
  it('is renamed in the braces of every face that stands it', async () => {
    const { tab } = await open()

    tab.namesField('Height', 'Shoulder')

    expect(tab.sheet().fields).toStrictEqual(['Shoulder', 'Life span'])
    expect(tab.sheet().faces[0]?.back).toBe('**Height:** {{Shoulder}}')
  })

  it('reaches the vault with the fields and the faces together', async () => {
    const { stencils, tab, written } = await open()

    tab.namesField('Height', 'Shoulder')
    await stencils.flush()

    expect(written).toStrictEqual(['Animal.md Shoulder, Life span | **Height:** {{Shoulder}}'])
  })

  it('writes nothing where nothing was touched', async () => {
    const { stencils, written } = await open()

    await stencils.flush()

    expect(written).toStrictEqual([])
  })
})

describe('a stencil whose file moved past what was read', () => {
  it('is overtaken once the write comes back saying the file changed', async () => {
    const { stencils, tab } = await open({ changed: true })

    tab.addsField('Weight')
    await stencils.flush()

    expect(tab.shown().state).toBe('overtaken')
  })

  it('keeps what the person wrote when they say so', async () => {
    const { stencils, tab, written } = await open({ changed: true })

    tab.addsField('Weight')
    await stencils.flush()
    tab.keep()
    await settles()

    expect(written).toHaveLength(2)
  })
})

describe('a stencil the vault refused', () => {
  it('says the note is not a stencil where that is what it is', async () => {
    const { tab } = await open({ refusal: 'notAStencil' })

    expect(tab.saying()).toBe(words.notAStencil)
    expect(tab.sheet().fields).toStrictEqual([])
  })

  it('says nothing where the stencil was read', async () => {
    const { tab } = await open()

    expect(tab.saying()).toBe('')
  })
})

describe('what is wrong with a stencil', () => {
  it('stands against the face it was read against', async () => {
    const { tab } = await open({
      problems: [
        { fault: 'faceMissingASide', card: null, face: 0, field: '', text: 'no back' },
      ],
    })

    const face = tab.sheet().faces[0]?.id ?? ''
    expect(tab.marks().at.get(face)).toStrictEqual(['no back'])
  })

  it('stands against the field it names where it stands against no face', async () => {
    const { tab } = await open({
      problems: [
        {
          fault: 'fieldDeclaredTwice',
          card: null,
          face: null,
          field: 'Height',
          text: 'declared twice',
        },
      ],
    })

    expect(tab.marks().fields.get('Height')).toStrictEqual(['declared twice'])
    expect(tab.marks().at.size).toBe(0)
  })

  it('is nothing at all where the vault reported none', async () => {
    const { tab } = await open()

    expect(tab.marks().at.size).toBe(0)
    expect(tab.marks().fields.size).toBe(0)
    expect(tab.marks().whole).toStrictEqual([])
  })
})
