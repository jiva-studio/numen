/**
 * What the vault says, read by the client the schema generates.
 *
 * `vault.test.ts` hands the mapping plain objects of its own making. This one
 * answers the window over the transport, so the message it reads is the one the
 * generated code builds: a field left at its default is one the answer does not
 * carry, and an enum arrives as the number the schema gave it.
 */
import { describe, expect, it, vi } from 'vitest'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

/** What the application answers with, in the words the schema writes it in. */
const answerWith = (answer: unknown) =>
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(JSON.stringify(answer), { headers: { 'content-type': 'application/json' } }),
    ),
  )

const { core } = await import('./index')

describe('a name the vault answers with', () => {
  it('is an ordinary note where the answer carries no kind at all', async () => {
    answerWith({ found: [{ note: { path: 'Entropy.md', title: 'Entropy' }, at: [] }] })

    expect((await core.names('ent', 8))[0]?.type).toBe('note')
  })

  it('is drawn as whichever of four the answer names', async () => {
    answerWith({
      found: [
        { note: { path: 'Ants.md', title: 'Ants' }, at: [], type: '' },
        { note: { path: 'Animals.md', title: 'Animals' }, at: [], type: 'deck' },
        { note: { path: 'Animal.md', title: 'Animal' }, at: [], type: 'stencil' },
        { note: { path: 'Daily.md', title: 'Daily' }, at: [], type: 'preset' },
      ],
    })

    expect((await core.names('an', 8)).map((one) => one.type)).toEqual([
      'note',
      'deck',
      'stencil',
      'preset',
    ])
  })

  it('is an ordinary note where the answer names a kind this window has no word for', async () => {
    answerWith({ found: [{ note: { path: 'Later.md', title: 'Later' }, at: [], type: 'later' }] })

    expect((await core.names('la', 8))[0]?.type).toBe('note')
  })
})

describe('a passage the vault answers with', () => {
  it('carries the kind of the note it was read out of', async () => {
    answerWith({
      found: [
        {
          path: 'Daily.md',
          note: { path: 'Daily.md', title: 'Daily' },
          at: [],
          type: 'preset',
        },
        { path: 'Ants.md', note: { path: 'Ants.md', title: 'Ants' }, at: [] },
      ],
    })

    const found = await core.search('an', 'words', 8)
    expect(found.map((one) => one.type)).toEqual(['preset', 'note'])
    expect(found.map((one) => one.isNote)).toEqual([true, true])
  })

  it('is no note at all where the answer names none', async () => {
    answerWith({ found: [{ path: 'library/mahabharata.epub', text: 'war', at: [] }] })

    const found = await core.search('war', 'words', 8)
    expect(found[0]?.isNote).toBe(false)
    expect(found[0]?.type).toBe('note')
  })

  it('carries what the vault holds at its path, whichever source that is', async () => {
    answerWith({
      found: [
        { path: 'Ants.md', note: { path: 'Ants.md' }, at: [], kind: 'SOURCE_KIND_NOTE' },
        { path: 'library/mahabharata.epub', at: [], kind: 'SOURCE_KIND_BOOK' },
        { path: 'talks/730709BG.LON.mp3', at: [], kind: 'SOURCE_KIND_RECORDING' },
      ],
    })

    expect((await core.search('war', 'words', 8)).map((one) => one.kind)).toEqual([
      'note',
      'book',
      'recording',
    ])
  })

  it('holds no source where the answer names none, or one this window cannot read', async () => {
    answerWith({
      found: [
        { path: 'notes.txt', at: [] },
        { path: 'later.xyz', at: [], kind: 9 },
      ],
    })

    expect((await core.search('war', 'words', 8)).map((one) => one.kind)).toEqual([
      'other',
      'other',
    ])
  })
})

describe('a neighbourhood the vault answers with', () => {
  it('is drawn around the note in focus, with each note in the relation it holds', async () => {
    answerWith({
      focus: { path: 'Rota.md', title: 'The rota' },
      focusType: 'deck',
      related: [
        {
          note: { path: 'Allotments.md', title: 'Allotments' },
          relation: 'RELATION_PARENT',
          label: 'part of',
          reverseLabel: 'contains rota',
          isMutual: true,
        },
      ],
    })

    const around = await core.neighbourhood('Rota.md')
    expect(around.focus).toEqual({ path: 'Rota.md', title: 'The rota' })
    expect(around.focusType).toBe('deck')
    expect(around.related).toEqual([
      {
        path: 'Allotments.md',
        title: 'Allotments',
        type: 'note',
        relation: 'parent',
        label: 'part of',
        reverseLabel: 'contains rota',
        through: '',
        isMutual: true,
      },
    ])
  })

  it('leaves out a note it has no relation for, and one the answer names no note at', async () => {
    answerWith({
      focus: { path: 'Rota.md', title: 'The rota' },
      related: [
        { note: { path: 'Odd.md', title: 'Odd' }, relation: 'RELATION_UNSPECIFIED' },
        { relation: 'RELATION_CHILD' },
      ],
    })

    expect((await core.neighbourhood('Rota.md')).related).toEqual([])
  })

  it('relies on nothing where the vault no longer holds the note', async () => {
    answerWith({ related: [] })

    expect((await core.neighbourhood('Gone.md')).focus.path).toBe('')
  })
})
