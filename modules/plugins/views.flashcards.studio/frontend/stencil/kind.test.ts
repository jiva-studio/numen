/**
 * What a stencil tab holds, asked without a screen: what it reads, what it
 * writes back, and where what is wrong with it stands.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/wire'
import type { ErrorCode } from '@numen/wire'
import type { Cards, VaultFace, DeckProblem, FieldRenameResult } from '../cards'
import { createMockBufferHost, createMockTabWindow } from '@numen/plugin-sdk/testing'
import { ERRORS } from '@numen/wire'
import { STENCIL, useStencilTabs, type StencilTabState } from './kind'
import { WORDS as words } from '../cards'

/** A moment for whatever the tab asked the vault for to come back. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const FACES: readonly VaultFace[] = [
  { name: 'Recognise', preamble: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
]

/** A vault holding one stencil, writing down every write it was asked for. */
const vault = (
  answers: {
    error?: ErrorCode
    problems?: readonly DeckProblem[]
    isChanged?: boolean
    /** What renaming a field comes back with, where a test wants another answer. */
    renaming?: FieldRenameResult
    /** The vault is out of reach, and a read of the stencil reaches nothing. */
    isUnreachable?: boolean
    /** What a write of the stencil fails for. */
    wrote?: ErrorCode
  } = {},
) => {
  const content: string[] = []
  /** Each field rename the tab asked the vault for. */
  const renamed: string[] = []
  let fields: readonly string[] = ['Height', 'Life span']
  let faces: readonly VaultFace[] = FACES

  const core: Cards = {
    stencils: async () => ({ stencils: [], held: 0 }),
    createDeck: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    createStencil: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    // The vault writes the name wherever it stands: in the fields, and in the
    // braces of every face.
    renameField: async (path, field, observed) => {
      const { from, to } = field
      renamed.push(`${path} ${from} ${to} ${observed ?? '—'}`)
      if (answers.renaming) return answers.renaming
      fields = fields.map((one) => (one === from ? to : one))
      const rewriteBraces = (text: string) => text.split(`{{${from}}}`).join(`{{${to}}}`)
      faces = faces.map((face) => ({
        ...face,
        front: rewriteBraces(face.front),
        back: rewriteBraces(face.back),
      }))
      return asValue({ decks: [], cards: 0, notWritten: [], at: 'renamed' })
    },
    readDeck: async () => asFailure({ code: 'missing' as const, bound: 0 }),
    writeDeck: async () => asValue({ at: '' }),
    readStencil: async (path) => {
      if (answers.isUnreachable) throw new Error('out of reach')
      if (answers.error) return asFailure(answers.error)
      return asValue({
        stencil: {
          path,
          title: 'Animal',
          fields,
          preamble: '',
          faces,
          tail: '',
          problems: answers.problems ?? [],
        },
        at: 'read',
      })
    },
    writeStencil: async (path, stencil, _seen) => {
      content.push(
        `${path} ${stencil.fields.join(', ') || '—'} | ${stencil.faces.map((one) => one.back).join(' ')}`,
      )
      if (answers.wrote) return asFailure(answers.wrote)
      if (answers.isChanged) return asFailure('changed' as const)
      fields = stencil.fields
      faces = stencil.faces
      return asValue({ at: 'written' })
    },
  }

  return {
    core,
    writtenFiles: content,
    renamed,
    /** The file written from somewhere else, which the next read answers with. */
    setFaces: (next: readonly VaultFace[]) => {
      faces = next
    },
  }
}

/** A window with one stencil open on a file, and what that tab holds. */
const open = async (answers: Parameters<typeof vault>[0] = {}, path = 'Animal.md') => {
  const one = vault(answers)
  const windowTabs = createMockTabWindow()
  const buffers = createMockBufferHost()
  /** Everything the window was given to say about this stencil. */
  const messages: string[] = []
  const stencils = useStencilTabs(one.core, {
    buffers,
    closeTab: windowTabs.finishClose,
    writeMessage: (text) => void messages.push(text),
  })
  windowTabs.register(stencils.kind)
  const id = await windowTabs.openTab(STENCIL, path)
  await settle()
  const tab = windowTabs.getTabState<StencilTabState>(id) as StencilTabState
  return { ...one, windowTabs, buffers, stencils, id, tab, messages }
}

describe('a stencil opened', () => {
  it('draws the fields in the order a person is requested for them', async () => {
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
    const { stencils, tab, writtenFiles } = await open()

    tab.renameField('Height', 'Shoulder')
    await settle()
    await stencils.store.flush()

    expect(writtenFiles).toStrictEqual([])
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
    const { tab, messages } = await open({
      renaming: asValue({
        decks: ['Animals.md', 'More.md'],
        cards: 3,
        notWritten: [],
        at: 'renamed',
      }),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toContain(words.renamed(3, 2))
  })

  it('says which decks keep the old heading, which nothing else would tell', async () => {
    const { tab, messages } = await open({
      renaming: asValue({
        decks: ['Animals.md'],
        cards: 1,
        notWritten: [{ path: 'Broken.md', text: 'the frontmatter cannot be read' }],
        at: 'renamed',
      }),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toContain(words.notWritten(['Broken.md']))
  })

  it('says the error, and says nothing of decks reached, where none was', async () => {
    const { tab, messages } = await open({
      renaming: asFailure('notAStencil' as const),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toStrictEqual([ERRORS.notAStencil])
  })

  it('says nothing was renamed where the file moved past the stencil that was read', async () => {
    const { tab, messages } = await open({
      renaming: asFailure('changed' as const),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toStrictEqual([words.notRenamed])
  })

  it('writes nothing where nothing was touched', async () => {
    const { stencils, writtenFiles } = await open()

    await stencils.store.flush()

    expect(writtenFiles).toStrictEqual([])
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
  it('is stale once the write arrives back saying the file changed', async () => {
    const { stencils, tab } = await open({ isChanged: true })

    tab.addField('Weight')
    await stencils.store.flush()

    expect(tab.note.value.state).toBe('stale')
  })

  it('keeps what the person wrote when they say so', async () => {
    const { stencils, tab, writtenFiles } = await open({ isChanged: true })

    tab.addField('Weight')
    await stencils.store.flush()
    tab.keepMine()
    await settle()

    expect(writtenFiles).toHaveLength(2)
  })
})

describe('a stencil the vault rejected', () => {
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
    const { tab } = await open({ isUnreachable: true })

    expect(tab.errorMessage.value).toBe(words.unreachable)
  })

  it('says the file could not be written, where that is what failed', async () => {
    const { stencils, tab } = await open({ wrote: 'unreadable' })

    tab.addField('Weight')
    await stencils.store.flushPending(stencils.store.getOpenIds()[0] ?? '')

    expect(tab.errorMessage.value).toBe(words.notSaved)
  })
})

describe('what is wrong with a stencil', () => {
  it('remains against the face it was read against', async () => {
    const { tab } = await open({
      problems: [{ kind: 'faceMissingASide', card: null, face: 0, field: '', text: 'no back' }],
    })

    const face = tab.stencil.value.faces[0]?.id ?? ''
    expect(tab.marks.value.at.get(face)).toStrictEqual(['no back'])
  })

  it('remains against the field it names at its current position against no face', async () => {
    const { tab } = await open({
      problems: [
        {
          kind: 'fieldDeclaredTwice',
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
    kind: 'faceMissingASide',
    card: null,
    face: 0,
    field: '',
    text: 'no back',
  }

  it('leaves what the editor is drawing standing, where the file reads the same', async () => {
    const one = await open({ problems: [sideless] })
    const was = { stencil: one.tab.stencil.value, marks: one.tab.marks.value }

    one.stencils.applyPathChanges(['Animal.md'])
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

    one.stencils.applyPathChanges(['Animal.md'])
    await settle()

    expect(one.tab.marks.value.at.get(face)).toStrictEqual(['no back'])
  })

  it('draws the file again where it was written from somewhere else', async () => {
    const one = await open()
    const was = one.tab.stencil.value

    one.setFaces([{ name: 'Recall', preamble: '', front: '{{Height}}', back: '{{Life span}}' }])
    one.stencils.applyPathChanges(['Animal.md'])
    await settle()

    expect(one.tab.stencil.value.faces.map((face) => face.name)).toStrictEqual(['Recall'])
    expect(one.tab.stencil.value).not.toBe(was)
  })
})

describe('a stencil renamed under the window', () => {
  it('is the tab it has when it is requested for at the name it now carries', async () => {
    const one = await open()

    one.stencils.applyPathChanges(['Beast.md'], [{ from: 'Animal.md', to: 'Beast.md' }])
    await settle()
    await one.windowTabs.openTab(STENCIL, 'Beast.md')
    await settle()

    expect(one.stencils.store.getOpenIds()).toHaveLength(1)
    expect(one.windowTabs.each(STENCIL)).toHaveLength(1)
  })

  it('is called what the file was called, before the name it went to is read', async () => {
    const one = await open()

    one.stencils.applyPathChanges(['Beast.md'], [{ from: 'Animal.md', to: 'Beast.md' }])

    expect(one.stencils.getTitle('Beast.md')).toBe('Animal')
  })
})

describe('a stencil whose tab has gone', () => {
  it('is nothing the window still says a word about', async () => {
    const one = await open()
    expect(one.stencils.getTitle('Animal.md')).toBe('Animal')

    one.windowTabs.requestClose(one.id)
    await settle()

    expect(one.stencils.getTitle('Animal.md')).toBe('Animal.md')
  })
})

/** A stencil whose write failed because its file moved: the tab is stale. */
const openStale = async () => {
  const one = await open({ isChanged: true })
  one.tab.addField('Weight')
  await one.stencils.store.flush()
  return one
}

describe('a stencil tab as the window lists it', () => {
  it('is the identity it opened under', async () => {
    const { stencils } = await open()

    expect(stencils.kind.identity?.('anything')).toBe('anything')
  })

  it('names the file it remains at and what it is called', async () => {
    const { stencils, tab } = await open()

    expect(stencils.kind.getTarget?.(tab)).toStrictEqual({ path: 'Animal.md', title: 'Animal' })
  })

  it('names the file it remains at where it is requested what is open', async () => {
    const { stencils, tab } = await open()

    expect(stencils.kind.getOpenTab?.(tab)).toStrictEqual({ path: 'Animal.md' })
  })

  it('carries no mark while its file is read, and the mark of its state once stale', async () => {
    const clean = await open()
    expect(clean.stencils.kind.getMark?.(clean.tab)).toBeUndefined()

    const staleSession = await openStale()
    expect(staleSession.stencils.kind.getMark?.(staleSession.tab)).toBe('stale')
  })

  it('is closed by the stencil itself, and the window is told to leave it be', async () => {
    const { stencils, tab } = await open()

    expect(stencils.kind.onClose?.(tab, 'no such tab')).toBe(false)
  })
})

describe('what the window asks of a stencil tab', () => {
  it('is whether the file reads as it did, or moved past it', async () => {
    const clean = await open()
    expect(clean.stencils.kind.isAsking?.(clean.tab)).toBe(false)

    const staleSession = await openStale()
    expect(staleSession.stencils.kind.isAsking?.(staleSession.tab)).toBe(true)
  })

  it('is to write what is unwritten out, and answer once it is', async () => {
    const one = await open()
    one.tab.addField('Weight')

    await one.stencils.kind.flushPending?.(one.tab)

    expect(one.written).toStrictEqual([
      'Animal.md Height, Life span, Weight | **Height:** {{Height}}',
    ])
  })

  it('is to be written out with the other open files when the window quits', async () => {
    const one = await open()
    one.tab.addField('Weight')

    await one.buffers.flush()

    expect(one.written).toStrictEqual([
      'Animal.md Height, Life span, Weight | **Height:** {{Height}}',
    ])
  })

  it('is to make a stencil of the name and the folder, carrying the first field', async () => {
    const one = await open()
    const createdItem = vi.spyOn(one.core, 'createStencil')

    await one.stencils.kind.createFile?.('plants', 'Plant')

    expect(createdItem).toHaveBeenCalledWith('Plant', 'plants', [words.newField])
  })
})

describe('a stencil tab closed', () => {
  it('stays open, still called what it was, where the file moved past what it holds', async () => {
    const one = await openStale()

    one.tab.close(one.id)
    await settle()

    expect(one.windowTabs.each(STENCIL)).toHaveLength(1)
    expect(one.stencils.store.getOpenIds()).toStrictEqual([one.tab.id])
    expect(one.stencils.getTitle('Animal.md')).toBe('Animal')
  })

  it('is read afresh where the file is opened again', async () => {
    const one = await open()
    const was = one.tab.stencil.value.faces[0]?.id

    one.tab.close(one.id)
    await settle()
    const id = await one.windowTabs.openTab(STENCIL, 'Animal.md')
    await settle()
    const tab = one.windowTabs.getTabState<StencilTabState>(id) as StencilTabState

    expect(tab.stencil.value.faces[0]?.id).not.toBe(was)
  })

  it('lets go of its title while a tab relies on another file', async () => {
    const one = await open()
    await one.windowTabs.openTab(STENCIL, 'Plant.md')
    await settle()

    one.windowTabs.requestClose('stencil:Animal.md')
    await settle()

    expect(one.stencils.getTitle('Animal.md')).toBe('Animal.md')
  })

  it('leaves the title with the other tab standing on the same file', async () => {
    const one = await open()
    await one.windowTabs.openTab(STENCIL, 'Beast.md')
    await settle()
    one.stencils.applyPathChanges(['Animal.md'], [{ from: 'Beast.md', to: 'Animal.md' }])
    await settle()

    one.tab.close(one.id)
    await settle()

    expect(one.stencils.getTitle('Animal.md')).toBe('Animal')
  })
})

describe('a stencil taken from its file', () => {
  it('shows the file as it now reads, and is no longer stale', async () => {
    const one = await openStale()

    one.tab.takeFile()
    await settle()

    expect(one.tab.note.value.state).not.toBe('stale')
    expect(one.tab.stencil.value.fields).toStrictEqual(['Height', 'Life span'])
  })
})

describe('a stencil opened from a file', () => {
  it('remains at one tab however many times it is requested for before it is open', async () => {
    const one = await open()

    await one.windowTabs.openTab(STENCIL, 'Plant.md')
    await one.windowTabs.openTab(STENCIL, 'Plant.md')
    await settle()

    expect(one.stencils.store.getOpenIds()).toHaveLength(2)
    expect(one.windowTabs.each(STENCIL)).toHaveLength(2)
  })

  it('remains at the file again once its tab was closed and it is opened again', async () => {
    const one = await open()
    const first = await one.windowTabs.openTab(STENCIL, 'Plant.md')
    await settle()
    one.windowTabs.requestClose(first)
    await settle()

    const again = await one.windowTabs.openTab(STENCIL, 'Plant.md')
    await settle()

    expect(one.stencils.store.getPath(again)).toBe('Plant.md')
  })
})
