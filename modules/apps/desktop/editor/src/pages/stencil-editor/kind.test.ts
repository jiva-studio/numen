/**
 * What a stencil tab holds, asked without a screen: what it reads, what it
 * writes back, and where what is wrong with it stands.
 */
import { describe, expect, it } from 'vitest'
import type { ErrorCode } from '@/shared/errors'
import type { Cards, VaultFace, DeckProblem, FieldRenameResult } from '@/entities/deck'
import { fileOpeners } from '@/entities/tab'
import { useWindowTabs } from '@/entities/tab'
import { STENCIL } from '@/entities/tab'
import { ERRORS } from '@/shared/words'
import { useStencilTabs, type StencilTabState } from './kind'
import { WORDS as words } from '@/entities/deck'

/** The one place a file is opened from. Nothing here opens one. */
const tabOpeners = () => fileOpeners({ fileKinds: async () => new Map() })

/** A moment for whatever the tab asked the vault for to come back. */
const settle = () => new Promise((done) => setTimeout(done, 0))

const FACES: readonly VaultFace[] = [
  { name: 'Recognise', preamble: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
]

/** A vault holding one stencil, writing down every write it was asked for. */
const vault = (
  answers: {
    error?: ErrorCode
    problems?: readonly DeckProblem[]
    changed?: boolean
    /** What renaming a field comes back with, where a test wants another answer. */
    renaming?: FieldRenameResult
    /** The vault is out of reach, and a read of the stencil reaches nothing. */
    unreachable?: boolean
    /** What a write of the stencil is refused for. */
    wrote?: ErrorCode
  } = {},
) => {
  const written: string[] = []
  /** Each field rename the tab asked the vault for. */
  const renamed: string[] = []
  let fields: readonly string[] = ['Height', 'Life span']
  let faces: readonly VaultFace[] = FACES

  const core: Cards = {
    stencils: async () => ({ stencils: [], held: 0 }),
    createDeck: async (title) => ({ path: `${title}.md`, error: null }),
    createStencil: async (title) => ({ path: `${title}.md`, error: null }),
    // The vault writes the name wherever it stands: in the fields, and in the
    // braces of every face.
    renameField: async (path, from, to, seen) => {
      renamed.push(`${path} ${from} ${to} ${seen ?? '—'}`)
      if (answers.renaming) return answers.renaming
      fields = fields.map((one) => (one === from ? to : one))
      const rewriteBraces = (text: string) => text.split(`{{${from}}}`).join(`{{${to}}}`)
      faces = faces.map((face) => ({
        ...face,
        front: rewriteBraces(face.front),
        back: rewriteBraces(face.back),
      }))
      return { decks: [], cards: 0, notWritten: [], error: null, changed: false, at: 'renamed' }
    },
    readDeck: async () => ({ deck: null, error: 'missing', at: '', bound: 0 }),
    writeDeck: async () => ({ error: null, changed: false, at: '', bound: 0 }),
    readStencil: async (path) => {
      if (answers.unreachable) throw new Error('out of reach')
      if (answers.error) return { stencil: null, error: answers.error, at: '' }
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
        error: null,
        at: 'read',
      }
    },
    writeStencil: async (path, wrote, drew) => {
      written.push(
        `${path} ${wrote.join(', ') || '—'} | ${drew.faces.map((one) => one.back).join(' ')}`,
      )
      if (answers.wrote) return { error: answers.wrote, changed: false, at: '' }
      if (answers.changed) return { error: null, changed: true, at: '' }
      fields = wrote
      faces = drew.faces
      return { error: null, changed: false, at: 'written' }
    },
  }

  return {
    core,
    written,
    renamed,
    /** The file written from somewhere else, which the next read answers with. */
    holds: (next: readonly VaultFace[]) => {
      faces = next
    },
  }
}

/** A window with one stencil open on a file, and what that tab holds. */
const open = async (answers: Parameters<typeof vault>[0] = {}, path = 'Animal.md') => {
  const one = vault(answers)
  const held = useWindowTabs()
  /** Everything the window was given to say about this stencil. */
  const said: string[] = []
  const road = tabOpeners()
  const stencils = useStencilTabs(one.core, held.handle, road, (text) => void said.push(text))
  held.registerKinds([stencils.kind])
  const id = await held.openTabOfKind(STENCIL, path)
  await settle()
  const tab = held.handle.getTabState<StencilTabState>(STENCIL, id) as StencilTabState
  return { ...one, held, road, stencils, id, tab, said }
}

describe('a stencil opened', () => {
  it('draws the fields in the order a person is asked for them', async () => {
    const { tab } = await open()

    expect(tab.stencil.value.fields).toStrictEqual(['Height', 'Life span'])
  })

  it('gives every face an identity, which the file carries none of', async () => {
    const { tab } = await open()

    expect(tab.stencil.value.faces[0]?.id).toBeTruthy()
    expect(tab.stencil.value.faces[0]?.name).toBe('Recognise')
  })

  it('is called what the file is called', async () => {
    const { stencils, tab } = await open()

    expect(stencils.kind.getTitle?.(tab)).toBe('Animal')
  })
})

describe('a field renamed in a stencil', () => {
  it('is the vault that renames it, presenting the file the tab read', async () => {
    const { tab, renamed } = await open()

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(renamed).toStrictEqual(['Animal.md Height Shoulder read'])
  })

  it('writes the stencil from the tab nowhere, so nothing goes over the vault', async () => {
    const { stencils, tab, written } = await open()

    tab.renameField('Height', 'Shoulder')
    await settle()
    await stencils.flush()

    expect(written).toStrictEqual([])
  })

  it('shows the fields and the braces as the vault left them', async () => {
    const { tab } = await open()

    tab.renameField('Height', 'Shoulder')
    await settle()
    await settle()

    expect(tab.stencil.value.fields).toStrictEqual(['Shoulder', 'Life span'])
    expect(tab.stencil.value.faces[0]?.back).toBe('**Height:** {{Shoulder}}')
  })

  it('asks for nothing where the name is the one the field carries', async () => {
    const { tab, renamed } = await open()

    tab.renameField('Height', 'Height')
    tab.renameField('Height', '')
    await settle()

    expect(renamed).toStrictEqual([])
  })

  it('says how far the new name reached', async () => {
    const { tab, said } = await open({
      renaming: {
        decks: ['Animals.md', 'More.md'],
        cards: 3,
        notWritten: [],
        error: null,
        changed: false,
        at: 'renamed',
      },
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(said).toContain(words.renamed(3, 2))
  })

  it('says which decks keep the old heading, which nothing else would tell', async () => {
    const { tab, said } = await open({
      renaming: {
        decks: ['Animals.md'],
        cards: 1,
        notWritten: [{ path: 'Broken.md', text: 'the frontmatter cannot be read' }],
        error: null,
        changed: false,
        at: 'renamed',
      },
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(said).toContain(words.notWritten(['Broken.md']))
  })

  it('says the error, and says nothing of decks reached, where none was', async () => {
    const { tab, said } = await open({
      renaming: {
        decks: [],
        cards: 0,
        notWritten: [],
        error: 'notAStencil',
        changed: false,
        at: '',
      },
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(said).toStrictEqual([ERRORS.notAStencil])
  })

  it('says nothing was renamed where the file moved past the stencil that was read', async () => {
    const { tab, said } = await open({
      renaming: {
        decks: ['Animals.md'],
        cards: 3,
        notWritten: [],
        error: null,
        changed: true,
        at: '',
      },
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(said).toStrictEqual([words.notRenamed])
  })

  it('writes nothing where nothing was touched', async () => {
    const { stencils, written } = await open()

    await stencils.flush()

    expect(written).toStrictEqual([])
  })
})

describe('a field carried in a stencil', () => {
  it('lands where it was let go', async () => {
    const { tab } = await open()

    tab.moveField('Life span', null)

    expect(tab.stencil.value.fields).toStrictEqual(['Height', 'Life span'])
  })

  it('leaves the first field first, wherever it was let go', async () => {
    const { tab } = await open()

    tab.moveField('Height', null)

    expect(tab.stencil.value.fields).toStrictEqual(['Height', 'Life span'])
  })

  it('lands nothing above the first field', async () => {
    const { tab } = await open()

    tab.addField('Weight')
    tab.moveField('Weight', 'Height')

    expect(tab.stencil.value.fields).toStrictEqual(['Height', 'Life span', 'Weight'])
  })
})

describe('a stencil whose file moved past what was read', () => {
  it('is stale once the write comes back saying the file changed', async () => {
    const { stencils, tab } = await open({ changed: true })

    tab.addField('Weight')
    await stencils.flush()

    expect(tab.note.value.state).toBe('stale')
  })

  it('keeps what the person wrote when they say so', async () => {
    const { stencils, tab, written } = await open({ changed: true })

    tab.addField('Weight')
    await stencils.flush()
    tab.keepMine()
    await settle()

    expect(written).toHaveLength(2)
  })
})

describe('a stencil the vault refused', () => {
  it('says the note is not a stencil where that is what it is', async () => {
    const { tab } = await open({ error: 'notAStencil' })

    expect(tab.errorMessage.value).toBe(words.notAStencil)
    expect(tab.stencil.value.fields).toStrictEqual([])
  })

  it('says nothing where the stencil was read', async () => {
    const { tab } = await open()

    expect(tab.errorMessage.value).toBe('')
  })

  it('says the vault could not be reached, where the read reached nothing', async () => {
    const { tab } = await open({ unreachable: true })

    expect(tab.errorMessage.value).toBe(words.unreachable)
  })

  it('says the file could not be written, where that is what was refused', async () => {
    const { stencils, tab } = await open({ wrote: 'unreadable' })

    tab.addField('Weight')
    await stencils.kept.settle(stencils.all()[0] ?? '')

    expect(tab.errorMessage.value).toBe(words.notSaved)
  })
})

describe('what is wrong with a stencil', () => {
  it('stands against the face it was read against', async () => {
    const { tab } = await open({
      problems: [
        { fault: 'faceMissingASide', card: null, face: 0, field: '', text: 'no back' },
      ],
    })

    const face = tab.stencil.value.faces[0]?.id ?? ''
    expect(tab.marks.value.at.get(face)).toStrictEqual(['no back'])
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

    expect(tab.marks.value.fields.get('Height')).toStrictEqual(['declared twice'])
    expect(tab.marks.value.at.size).toBe(0)
  })

  it('is nothing at all where the vault reported none', async () => {
    const { tab } = await open()

    expect(tab.marks.value.at.size).toBe(0)
    expect(tab.marks.value.fields.size).toBe(0)
    expect(tab.marks.value.whole).toStrictEqual([])
  })
})

describe('a stencil read again under the window', () => {
  // The file is read wrong in a way that stands against one face, so a mark
  // that went is a mark this can see going.
  const sideless: DeckProblem = {
    fault: 'faceMissingASide',
    card: null,
    face: 0,
    field: '',
    text: 'no back',
  }

  it('leaves what the editor is drawing standing, where the file reads the same', async () => {
    const one = await open({ problems: [sideless] })
    const was = { stencil: one.tab.stencil.value, marks: one.tab.marks.value }

    one.stencils.changed(['Animal.md'])
    await settle()

    // Each of them the same thing, and not merely a thing that reads the same:
    // a field under the keyboard is redrawn by anything else.
    expect(one.tab.stencil.value).toBe(was.stencil)
    expect(one.tab.marks.value).toBe(was.marks)
  })

  it('leaves each mark standing on the face the editor is drawing', async () => {
    const one = await open({ problems: [sideless] })
    // The face the editor is drawing, under the identity it was drawn with.
    const face = one.tab.stencil.value.faces[0]?.id ?? ''

    one.stencils.changed(['Animal.md'])
    await settle()

    expect(one.tab.marks.value.at.get(face)).toStrictEqual(['no back'])
  })

  it('draws the file again where it was written from somewhere else', async () => {
    const one = await open()
    const was = one.tab.stencil.value

    one.holds([{ name: 'Recall', preamble: '', front: '{{Height}}', back: '{{Life span}}' }])
    one.stencils.changed(['Animal.md'])
    await settle()

    expect(one.tab.stencil.value.faces.map((face) => face.name)).toStrictEqual(['Recall'])
    expect(one.tab.stencil.value).not.toBe(was)
  })
})

describe('a stencil renamed under the window', () => {
  it('is the tab it has when it is asked for at the name it now carries', async () => {
    const one = await open()

    one.stencils.changed(['Beast.md'], [{ from: 'Animal.md', to: 'Beast.md' }])
    await settle()
    one.road.openNewFile('Beast.md', '', 'stencil')
    await settle()

    expect(one.stencils.all()).toHaveLength(1)
    expect(one.held.handle.each(STENCIL)).toHaveLength(1)
  })

  it('is called what the file was called, before the name it went to is read', async () => {
    const one = await open()

    one.stencils.changed(['Beast.md'], [{ from: 'Animal.md', to: 'Beast.md' }])

    expect(one.stencils.getTitle('Beast.md')).toBe('Animal')
  })
})

describe('a stencil whose tab has gone', () => {
  it('is nothing the window still says a word about', async () => {
    const one = await open()
    expect(one.stencils.getTitle('Animal.md')).toBe('Animal')

    one.held.shut(one.id)
    await settle()

    expect(one.stencils.getTitle('Animal.md')).toBe('Animal.md')
  })
})
