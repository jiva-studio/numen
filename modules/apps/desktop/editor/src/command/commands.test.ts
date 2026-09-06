/**
 * What is offered over what is in front, and what a command carries out of it.
 *
 * The table is read here without a palette: which group a command stands in,
 * which of them another command's row reaches, and what the note a search did
 * not find is offered as.
 */
import { describe, expect, it } from 'vitest'
import {
  asksCommands,
  commandsOf,
  creates,
  MAKING,
  offering,
  overNote,
  type CommandTarget,
} from './commands'
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

describe('the character that means the commands', () => {
  it('is one typed into a field holding nothing', () => {
    expect(asksCommands('', '>')).toBe(true)
  })

  it('is not one typed into a field holding words, so a search for one stands', () => {
    expect(asksCommands('foo', '>foo')).toBe(false)
    expect(asksCommands('', '>foo')).toBe(false)
    expect(asksCommands('>', '>>')).toBe(false)
  })
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

describe('the commands over a note', () => {
  it('leave out the ones another command reaches on its own row', () => {
    const over = overNote(commandsOf(words)).map((one) => one.id)

    expect(over).not.toContain('beside')
    expect(over).not.toContain('destroy')
  })
})

describe('the commands over the vaults an installation holds', () => {
  it('are offered in the group of the vault in front', () => {
    const over = commandsOf(words)
      .filter((one) => one.group === 'vault')
      .map((one) => one.id)

    expect(over).toStrictEqual([
      'first',
      'goto',
      'openVault',
      'newVault',
      'renameVault',
      'forgetVault',
      'eraseVault',
    ])
  })
})
