/**
 * What a stencil tab holds, asked without a screen: what it reads, what it
 * writes back, and where what is wrong with it stands.
 */
import { describe, expect, it } from 'vitest'
import type { Cards, Faced, Problem, Refused, Renaming } from '../core'
import { putting } from '../putting'
import { windowing } from '../windowing'
import { STENCIL } from '../workspace'
import { REFUSED } from '../words'
import { stencilling, type Held } from './stencil'
import { WORDS as words } from './words'

/** The one place a file is opened from. Nothing here opens one. */
const puts = () => putting({ standing: async () => new Map() })

/** A moment for whatever the tab asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

const FACES: readonly Faced[] = [
  { name: 'Recognise', lead: '', front: '{{title}}', back: '**Height:** {{Height}}' },
]

/** A vault holding one stencil, writing down every write it was asked for. */
const vault = (
  answers: {
    refusal?: Refused
    problems?: readonly Problem[]
    changed?: boolean
    /** What renaming a field comes back with, where a test wants another answer. */
    renaming?: Renaming
  } = {},
) => {
  const written: string[] = []
  /** Each field rename the tab asked the vault for. */
  const renamed: string[] = []
  let fields: readonly string[] = ['Height', 'Life span']
  let faces: readonly Faced[] = FACES

  const core: Cards = {
    stencils: async () => ({ stencils: [], held: 0 }),
    makeDeck: async (title) => ({ path: `${title}.md`, refusal: null }),
    makeStencil: async (title) => ({ path: `${title}.md`, refusal: null }),
    // The vault writes the name wherever it stands: in the fields, and in the
    // braces of every face.
    renameField: async (path, from, to, seen) => {
      renamed.push(`${path} ${from} ${to} ${seen ?? '—'}`)
      if (answers.renaming) return answers.renaming
      fields = fields.map((one) => (one === from ? to : one))
      const braces = (text: string) => text.split(`{{${from}}}`).join(`{{${to}}}`)
      faces = faces.map((face) => ({
        ...face,
        front: braces(face.front),
        back: braces(face.back),
      }))
      return { decks: [], cards: 0, notWritten: [], refusal: null, changed: false, at: 'renamed' }
    },
    readDeck: async () => ({ deck: null, refusal: 'missing', at: '', bound: 0 }),
    writeDeck: async () => ({ refusal: null, changed: false, at: '', bound: 0 }),
    readStencil: async (path) => {
      if (answers.refusal) return { stencil: null, refusal: answers.refusal, at: '' }
      return {
        stencil: {
          path,
          title: 'Animal',
          fields,
          preamble: '',
          faces,
          tail: '',
          problems: answers.problems ?? [],
        },
        refusal: null,
        at: 'read',
      }
    },
    writeStencil: async (path, wrote, drew) => {
      written.push(
        `${path} ${wrote.join(', ') || '—'} | ${drew.faces.map((one) => one.back).join(' ')}`,
      )
      if (answers.changed) return { refusal: null, changed: true, at: '' }
      fields = wrote
      faces = drew.faces
      return { refusal: null, changed: false, at: 'written' }
    },
  }

  return {
    core,
    written,
    renamed,
    /** The file written from somewhere else, which the next read answers with. */
    holds: (next: readonly Faced[]) => {
      faces = next
    },
  }
}

/** A window with one stencil open on a file, and what that tab holds. */
const open = async (answers: Parameters<typeof vault>[0] = {}, path = 'Animal.md') => {
  const one = vault(answers)
  const held = windowing()
  /** Everything the window was given to say about this stencil. */
  const said: string[] = []
  const stencils = stencilling(one.core, held.host, puts(), (text) => void said.push(text))
  held.declares([stencils.kind])
  const id = await held.opens(STENCIL, path)
  await settles()
  const tab = held.host.holds<Held>(STENCIL, id) as Held
  return { ...one, held, stencils, id, tab, said }
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
  it('is the vault that renames it, presenting the file the tab read', async () => {
    const { tab, renamed } = await open()

    tab.namesField('Height', 'Shoulder')
    await settles()

    expect(renamed).toStrictEqual(['Animal.md Height Shoulder read'])
  })

  it('writes the stencil from the tab nowhere, so nothing goes over the vault', async () => {
    const { stencils, tab, written } = await open()

    tab.namesField('Height', 'Shoulder')
    await settles()
    await stencils.flush()

    expect(written).toStrictEqual([])
  })

  it('shows the fields and the braces as the vault left them', async () => {
    const { tab } = await open()

    tab.namesField('Height', 'Shoulder')
    await settles()
    await settles()

    expect(tab.sheet().fields).toStrictEqual(['Shoulder', 'Life span'])
    expect(tab.sheet().faces[0]?.back).toBe('**Height:** {{Shoulder}}')
  })

  it('asks for nothing where the name is the one the field carries', async () => {
    const { tab, renamed } = await open()

    tab.namesField('Height', 'Height')
    tab.namesField('Height', '')
    await settles()

    expect(renamed).toStrictEqual([])
  })

  it('says how far the new name reached', async () => {
    const { tab, said } = await open({
      renaming: {
        decks: ['Animals.md', 'More.md'],
        cards: 3,
        notWritten: [],
        refusal: null,
        changed: false,
        at: 'renamed',
      },
    })

    tab.namesField('Height', 'Shoulder')
    await settles()

    expect(said).toContain(words.renamed(3, 2))
  })

  it('says which decks keep the old heading, which nothing else would tell', async () => {
    const { tab, said } = await open({
      renaming: {
        decks: ['Animals.md'],
        cards: 1,
        notWritten: [{ path: 'Broken.md', text: 'the frontmatter cannot be read' }],
        refusal: null,
        changed: false,
        at: 'renamed',
      },
    })

    tab.namesField('Height', 'Shoulder')
    await settles()

    expect(said).toContain(words.notWritten(['Broken.md']))
  })

  it('says the refusal, and says nothing of decks reached, where none was', async () => {
    const { tab, said } = await open({
      renaming: {
        decks: [],
        cards: 0,
        notWritten: [],
        refusal: 'notAStencil',
        changed: false,
        at: '',
      },
    })

    tab.namesField('Height', 'Shoulder')
    await settles()

    expect(said).toStrictEqual([REFUSED.notAStencil])
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

describe('a stencil read again under the window', () => {
  it('leaves what the editor is drawing standing, where the file reads the same', async () => {
    const one = await open()
    const was = { sheet: one.tab.sheet(), marks: one.tab.marks() }

    one.stencils.changed(['Animal.md'])
    await settles()

    // Each of them the same thing, and not merely a thing that reads the same:
    // a field under the keyboard is redrawn by anything else.
    expect(one.tab.sheet()).toBe(was.sheet)
    expect(one.tab.marks()).toBe(was.marks)
  })

  it('draws the file again where it was written from somewhere else', async () => {
    const one = await open()
    const was = one.tab.sheet()

    one.holds([{ name: 'Recall', lead: '', front: '{{Height}}', back: '{{title}}' }])
    one.stencils.changed(['Animal.md'])
    await settles()

    expect(one.tab.sheet().faces.map((face) => face.name)).toStrictEqual(['Recall'])
    expect(one.tab.sheet()).not.toBe(was)
  })
})
