/**
 * What the vault returns, as read by the client the schema generates.
 *
 * `vault.test.ts` passes the mapping plain objects of its own construction. This test
 * serves the window over the transport, so the message it reads is the one the
 * generated code builds: a field left at its default is not carried in the response,
 * and an enum arrives as the number the schema assigns.
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

describe('vault name responses', () => {
  it('is an ordinary note where the answer carries no kind at all', async () => {
    answerWith({ found: [{ note: { path: 'Entropy.md', title: 'Entropy' }, at: [] }] })

    expect((await core.names('ent', 8))[0]?.type).toBe('note')
  })

  it('is drawn as whichever of four the answer names', async () => {
    answerWith({
      found: [
        { note: { path: 'Ants.md', title: 'Ants' }, at: [], type: '' },
        { note: { path: 'Animals.md', title: 'Animals' }, at: [], type: 'sample' },
        { note: { path: 'Animal.md', title: 'Animal' }, at: [], type: 'form' },
        { note: { path: 'Daily.md', title: 'Daily' }, at: [], type: 'sheet' },
      ],
    })

    expect((await core.names('an', 8)).map((one) => one.type)).toEqual([
      'note',
      'sample',
      'form',
      'sheet',
    ])
  })

  it('carries the type a plugin declares as the answer names it', async () => {
    answerWith({ found: [{ note: { path: 'Later.md', title: 'Later' }, at: [], type: 'later' }] })

    expect((await core.names('la', 8))[0]?.type).toBe('later')
  })
})

describe('vault passage responses', () => {
  it('carries the kind of the note it was read out of', async () => {
    answerWith({
      found: [
        {
          path: 'Daily.md',
          note: { path: 'Daily.md', title: 'Daily' },
          at: [],
          type: 'sheet',
        },
        { path: 'Ants.md', note: { path: 'Ants.md', title: 'Ants' }, at: [] },
      ],
    })

    const found = await core.search('an', 'words', 8)
    expect(found.map((one) => one.type)).toEqual(['sheet', 'note'])
    expect(found.map((one) => one.isNote)).toEqual([true, true])
  })

  it('is no note at all where the answer names none', async () => {
    answerWith({ found: [{ path: 'library/mahabharata.epub', text: 'war', at: [] }] })

    const found = await core.search('war', 'words', 8)
    expect(found[0]?.isNote).toBe(false)
    expect(found[0]?.type).toBe('note')
  })

  it('carries the selector of a hit in a file that is no note', async () => {
    answerWith({
      found: [
        { path: 'Ants.md', note: { path: 'Ants.md' }, at: [] },
        { path: 'library/a.bin', at: [], selector: '{"page":3}' },
      ],
    })

    expect((await core.search('war', 'words', 8)).map((one) => one.selector)).toEqual([
      null,
      { page: 3 },
    ])
  })

  it('holds no selector where the hit carries none or one that is not JSON', async () => {
    answerWith({
      found: [
        { path: 'a.bin', at: [], selector: '' },
        { path: 'b.bin', at: [], selector: '{page' },
      ],
    })

    expect((await core.search('war', 'words', 8)).map((one) => one.selector)).toEqual([null, null])
  })
})

describe('vault neighbourhood responses', () => {
  it('is drawn around the note in focus, with each note in the relation it holds', async () => {
    answerWith({
      focus: { path: 'Rota.md', title: 'The rota' },
      focusType: 'sample',
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

    const neighbourhood = await core.neighbourhood('Rota.md')
    expect(neighbourhood.focus).toEqual({ path: 'Rota.md', title: 'The rota' })
    expect(neighbourhood.focusType).toBe('sample')
    expect(neighbourhood.related).toEqual([
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
