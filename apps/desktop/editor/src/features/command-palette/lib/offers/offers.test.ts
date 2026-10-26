/**
 * What a search that turned up nothing puts in place of an answer, and what
 * the note it offers is made as.
 */
import { describe, expect, it } from 'vitest'
import { appendCreateOffer, createNoteInvocation, CREATE_NOTE_ID } from './offers'
import type { CommandTarget } from '../../types'
import { WORDS as words } from '@/shared/words'

/** What is in front, which a test moves under the commands. */
const front = (overrides: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  source: null,
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...overrides,
})

describe('a search that turned up nothing', () => {
  const createGroups = (items: number, isWorking = false) => [
    {
      id: 'names',
      title: 'Names',
      items: Array.from({ length: items }, (_, index) => ({ id: `${index}`, title: 'One' })),
      isWorking: isWorking,
    },
  ]

  it('offers to make the note that was looked for', () => {
    const proposed = appendCreateOffer(createGroups(0), { text: 'Entropy', words, target: front() })

    expect(proposed.at(-1)?.id).toBe(CREATE_NOTE_ID)
    expect(proposed.at(-1)?.items[0]?.title).toBe('Create a note called “Entropy”')
  })

  it('offers nothing while a group is still waiting on the vault', () => {
    expect(
      appendCreateOffer(createGroups(0, true), { text: 'Entropy', words, target: front() }),
    ).toHaveLength(1)
  })

  it('offers nothing where a group turned something up', () => {
    expect(
      appendCreateOffer(createGroups(1), { text: 'Entropy', words, target: front() }),
    ).toHaveLength(1)
  })

  it('offers nothing where there are no groups to have answered', () => {
    expect(appendCreateOffer([], { text: 'Entropy', words, target: front() })).toHaveLength(0)
  })

  it('offers nothing where nothing was looked for', () => {
    expect(
      appendCreateOffer(createGroups(0), { text: '   ', words, target: front() }),
    ).toHaveLength(1)
  })

  it('makes the note where the person is standing, under the words looked for', () => {
    expect(createNoteInvocation('creating', 'Entropy', front())).toStrictEqual({
      id: 'note',
      path: '',
      vault: { id: 'physics', name: 'Physics' },
      note: null,
      title: '',
      file: '',
      others: [],
      name: 'Entropy',
      kind: 'note',
      tab: 'tab',
    })
  })

  it('offers the relations of the note in front, and says which note that is', () => {
    const item = appendCreateOffer(createGroups(0), { text: 'Entropy', words, target: front() }).at(
      -1,
    )?.items[0]

    expect(item?.actions?.map((one) => one.id)).toStrictEqual([
      CREATE_NOTE_ID,
      'child',
      'parent',
      'jump',
    ])
    expect(item?.detail).toBe(front().title)
  })

  it('offers no relation where nothing in front is a note', () => {
    const item = appendCreateOffer(createGroups(0), {
      text: 'Entropy',
      words,
      target: front({ path: '', title: '' }),
    }).at(-1)?.items[0]

    expect(item?.actions?.map((one) => one.id)).toStrictEqual([CREATE_NOTE_ID])
    expect(item?.detail).toBeUndefined()
  })

  it('hangs the note off the one in front when a relation was chosen', () => {
    expect(createNoteInvocation('child', 'Entropy', front())).toMatchObject({
      id: 'child',
      path: front().path,
      name: 'Entropy',
    })
  })

  it('remains the note on its own where a relation has no note to hang it off', () => {
    expect(createNoteInvocation('child', 'Entropy', front({ path: '', title: '' }))).toMatchObject({
      id: 'note',
      path: '',
      name: 'Entropy',
    })
  })
})
