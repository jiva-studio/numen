/**
 * What a search that turned up nothing puts in place of an answer, and what
 * the note it offers is made as.
 */
import { describe, expect, it } from 'vitest'
import { creates, MAKING, offering } from './offers'
import type { CommandTarget } from './commands'
import { WORDS as words } from '../words'

/** What is in front, which a test moves under the commands. */
const front = (over: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  source: null,
  made: {},
  vault: { id: 'physics', name: 'Physics' },
  ready: true,
  ...over,
})

describe('a search that turned up nothing', () => {
  const groups = (items: number, working = false) => [
    { id: 'names', title: 'Names', items: Array.from({ length: items }, (_, at) => ({ id: `${at}`, title: 'One' })), working },
  ]

  it('offers to make the note that was looked for', () => {
    const offered = offering(groups(0), 'Entropy', words, front())

    expect(offered.at(-1)?.id).toBe(MAKING)
    expect(offered.at(-1)?.items[0]?.title).toBe('Create a note called “Entropy”')
  })

  it('offers nothing while a group is still waiting on the vault', () => {
    expect(offering(groups(0, true), 'Entropy', words, front())).toHaveLength(1)
  })

  it('offers nothing where a group turned something up', () => {
    expect(offering(groups(1), 'Entropy', words, front())).toHaveLength(1)
  })

  it('offers nothing where nothing was looked for', () => {
    expect(offering(groups(0), '   ', words, front())).toHaveLength(1)
  })

  it('makes the note where the person is standing, under the words looked for', () => {
    expect(creates('creating', 'Entropy', front())).toStrictEqual({
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

  it('offers the seats of the note in front, and says which note that is', () => {
    const item = offering(groups(0), 'Entropy', words, front()).at(-1)?.items[0]

    expect(item?.actions?.map((one) => one.id)).toStrictEqual([
      MAKING,
      'child',
      'parent',
      'jump',
    ])
    expect(item?.detail).toBe(front().title)
  })

  it('offers no seat where nothing in front is a note', () => {
    const item = offering(groups(0), 'Entropy', words, front({ path: '', title: '' })).at(-1)
      ?.items[0]

    expect(item?.actions?.map((one) => one.id)).toStrictEqual([MAKING])
    expect(item?.detail).toBeUndefined()
  })

  it('hangs the note off the one in front when a seat was chosen', () => {
    expect(creates('child', 'Entropy', front())).toMatchObject({
      id: 'child',
      path: front().path,
      name: 'Entropy',
    })
  })

  it('stands the note on its own where a seat has no note to hang it off', () => {
    expect(creates('child', 'Entropy', front({ path: '', title: '' }))).toMatchObject({
      id: 'note',
      path: '',
      name: 'Entropy',
    })
  })
})
