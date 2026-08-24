/**
 * What is offered over what is in front, and what a step asks for, without a
 * window.
 *
 * Two things are asked here that nothing else can ask: that a tab holding
 * something that is not a note is offered nothing to do to a note, and that
 * destroying is reached by typing the name of the note and by nothing else.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import {
  asksCommands,
  commanding,
  commandsOf,
  creates,
  MAKING,
  offering,
  overNote,
  type Where,
} from './commanding'
import type { Named } from './finding'
import { WORDS as words } from './words'

/** What is in front, which a test moves under the commands. */
const front = (over: Partial<Where> = {}): Where => ({
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  ready: true,
  ...over,
})

/** One name as the vault answers one. */
const name = (path: string, title: string, heading = ''): Named => ({
  path,
  title,
  heading,
  line: heading ? 4 : -1,
  at: [{ from: 0, to: 1 }],
})

/** The commands over what a test says is in front, asked without a hold. */
const asking = (over: Partial<Where> = {}, found: readonly Named[] = []) => {
  const at = ref(front(over))
  const asked: string[] = []
  const core = {
    names: async (query: string) => {
      asked.push(query)
      return found
    },
  }
  const commands = commanding(core, words, () => at.value, async () => {})
  commands.shows(true)
  return { commands, at, asked }
}

/** Every item drawn, band by band, under the band it stands in. */
const drawn = (bands: ReturnType<typeof asking>['commands']['bands']) =>
  Object.fromEntries(bands.value.map((band) => [band.id, band.items.map((item) => item.id)]))

/** What one band says while it holds nothing. */
const silence = (bands: ReturnType<typeof asking>['commands']['bands'], id: string) =>
  bands.value.find((band) => band.id === id)?.silence

describe('the commands as they open', () => {
  it('offers everything that can be done to what is in front, with nothing typed', () => {
    const { commands } = asking()

    expect(drawn(commands.bands)).toStrictEqual({
      note: ['read', 'travel', 'child', 'parent', 'jump', 'title', 'remove', 'ask', 'copy'],
      window: ['note', 'plex', 'agent', 'close', 'find'],
      vault: ['first', 'goto'],
    })
  })

  it('draws the command reached by Shift and Enter on the row that names it', () => {
    const { commands } = asking()
    const items = commands.bands.value[0]?.items ?? []

    expect(items.find((item) => item.id === 'read')?.actions?.map((one) => one.id)).toStrictEqual([
      'read',
      'beside',
    ])
    expect(items.find((item) => item.id === 'remove')?.actions?.map((one) => one.id)).toStrictEqual(
      ['remove', 'destroy'],
    )
  })

  it('says which key reaches a command away from the palette', () => {
    const { commands } = asking()
    const window = commands.bands.value[1]?.items ?? []

    expect(window.find((item) => item.id === 'find')?.keys).toBe(words.findKeys)
  })

  it('says what each command over the note is over', () => {
    const { commands } = asking()

    for (const item of commands.bands.value[0]?.items ?? []) expect(item.detail).toBe('Ontology')
  })

  it('keeps the commands the words typed leave, and lights where they stand', () => {
    const { commands } = asking()

    void commands.typing('child')

    expect(drawn(commands.bands)).toStrictEqual({ note: ['child'], window: [], vault: [] })
    expect(commands.bands.value[0]?.items[0]?.at).toStrictEqual([{ from: 4, to: 9 }])
  })
})

describe('what is in front', () => {
  it('offers nothing over a note when a document is in front', () => {
    const { commands } = asking({ kind: 'document', path: '', title: '' })

    expect(drawn(commands.bands).note).toStrictEqual([])
    expect(silence(commands.bands, 'note')).toBe(words.noNote)
  })

  it('offers nothing over a note when the tab in front holds nothing yet', () => {
    const { commands } = asking({ kind: null, path: '', title: '' })

    expect(drawn(commands.bands).note).toStrictEqual([])
  })

  it('offers nothing over a note when the plex is standing nowhere', () => {
    const { commands } = asking({ kind: 'plex', path: '', title: '' })

    expect(drawn(commands.bands).note).toStrictEqual([])
  })

  it('says the vault is still being read rather than quietly offering less', () => {
    const { commands } = asking({ ready: false })

    expect(drawn(commands.bands)).toStrictEqual({
      note: [],
      window: ['plex', 'agent', 'close', 'find'],
      vault: [],
    })
    expect(silence(commands.bands, 'note')).toBe(words.indexing)
  })

  it('offers no close where the window holds no tab at all', () => {
    const { commands } = asking({ tab: '' })

    expect(drawn(commands.bands).window).not.toContain('close')
  })

  it('follows what the person is looking at while it is open', () => {
    const { commands, at } = asking()

    at.value = front({ kind: 'document', path: '', title: '' })

    expect(drawn(commands.bands).note).toStrictEqual([])
  })
})

describe('a command that needs nothing', () => {
  it('is a deed the moment it is chosen, over the note in front', () => {
    const { commands } = asking()

    expect(commands.chose('read', 'read')).toStrictEqual({
      id: 'read',
      path: 'physics/Ontology.md',
      title: 'Ontology',
      name: '',
      kind: 'note',
      tab: 'tab',
    })
  })

  it('is the second one where Shift and Enter reached it', () => {
    const { commands } = asking()

    expect(commands.chose('read', 'beside')?.id).toBe('beside')
  })

  it('is nothing where what is in front does not offer it', () => {
    const { commands } = asking({ kind: 'document', path: '', title: '' })

    expect(commands.chose('read', 'read')).toBeNull()
  })
})

describe('a command that asks for a name', () => {
  it('stands on a step of its own, and says which step it is', () => {
    const { commands } = asking()

    expect(commands.asks('child', front())).toBeNull()
    expect(commands.crumb.value).toBe(words.child)
    expect(commands.bands.value.map((band) => band.id)).toStrictEqual(['naming'])
  })

  it('offers what was typed as the name, and nothing before anything is', () => {
    const { commands } = asking()
    commands.asks('child', front())

    expect(commands.bands.value[0]?.items).toStrictEqual([])

    void commands.typing('  Entropy  ')

    expect(commands.bands.value[0]?.items[0]?.title).toBe('Call it “Entropy”')
  })

  it('carries the name to the note it was asked over', () => {
    const { commands } = asking()
    commands.asks('child', front())
    void commands.typing('Entropy')

    expect(commands.chose('name', 'name')).toStrictEqual({
      id: 'child',
      path: 'physics/Ontology.md',
      title: 'Ontology',
      name: 'Entropy',
      kind: 'note',
      tab: 'tab',
    })
  })

  it('is nothing while nothing has been typed', () => {
    const { commands } = asking()
    commands.asks('child', front())

    expect(commands.chose('name', 'name')).toBeNull()
  })

  it('puts the title the note has now in the field, for a change of title', () => {
    const { commands } = asking()

    commands.asks('title', front())

    expect(commands.typed.value).toBe('Ontology')
  })
})

describe('removing a note', () => {
  it('asks once, naming the note and saying where it goes', () => {
    const { commands } = asking()

    commands.asks('remove', front())

    expect(commands.bands.value[0]?.items[0]?.title).toBe('Remove “Ontology”')
    expect(commands.bands.value[0]?.items[0]?.detail).toBe(words.trashed)
  })

  it('is a deed once that question is answered', () => {
    const { commands } = asking()
    commands.asks('remove', front())

    expect(commands.chose('yes', 'yes')?.id).toBe('remove')
  })
})

describe('destroying a note', () => {
  it('waits for the name of the note, and is not reachable before it', () => {
    const { commands } = asking()
    commands.asks('destroy', front())

    expect(commands.bands.value[0]?.items[0]?.disabled).toBe(true)
    expect(commands.chose('exactly', 'exactly')).toBeNull()

    void commands.typing('Ontolog')

    expect(commands.chose('exactly', 'exactly')).toBeNull()
  })

  it('goes once the name of the note is what was typed', () => {
    const { commands } = asking()
    commands.asks('destroy', front())
    void commands.typing('Ontology')

    expect(commands.bands.value[0]?.items[0]?.disabled).toBe(false)
    expect(commands.chose('exactly', 'exactly')?.id).toBe('destroy')
  })
})

describe('a command that asks for a note', () => {
  it('asks the vault for the names that match, and draws each note once', async () => {
    const { commands, asked } = asking({}, [
      name('physics/Entropy.md', 'Entropy'),
      name('physics/Entropy.md', 'Entropy', 'What it counts'),
      name('Order.md', 'Order'),
    ])
    commands.asks('goto', front())

    await commands.typing('en')

    expect(asked).toStrictEqual(['en'])
    expect(commands.bands.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics/Entropy.md',
      'Order.md',
    ])
  })

  it('carries the note that was picked, not the one that was in front', async () => {
    const { commands } = asking({}, [name('physics/Entropy.md', 'Entropy')])
    commands.asks('goto', front())
    await commands.typing('en')

    expect(commands.chose('physics/Entropy.md', 'pick')).toStrictEqual({
      id: 'goto',
      path: 'physics/Entropy.md',
      title: 'Entropy',
      name: '',
      kind: 'note',
      tab: 'tab',
    })
  })

  it('says the vault could not answer, in the window’s own words', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    const at = ref(front())
    const commands = commanding(
      { names: async () => Promise.reject(new Error('no model is set')) },
      words,
      () => at.value,
      async () => {},
    )
    commands.shows(true)
    commands.asks('goto', front())

    await commands.typing('en')

    expect(commands.bands.value[0]?.silence).toBe(words.notAsked)
  })
})

describe('leaving a step', () => {
  it('goes back to the commands, and the palette stays open', () => {
    const { commands } = asking()
    commands.asks('child', front())

    commands.leaves()

    expect(commands.open.value).toBe(true)
    expect(commands.bands.value.map((band) => band.id)).toStrictEqual(['note', 'window', 'vault'])
  })

  it('puts the commands away from the commands themselves', () => {
    const { commands } = asking()

    commands.leaves()

    expect(commands.open.value).toBe(false)
  })

  it('hands the field back to the search from the commands themselves', () => {
    const { commands } = asking()

    expect(commands.backs()).toBe(true)
    expect(commands.open.value).toBe(false)
  })

  it('keeps the field on a step, and drops the step', () => {
    const { commands } = asking()
    commands.asks('child', front())

    expect(commands.backs()).toBe(false)
    expect(commands.crumb.value).toBe(words.command)
  })

  it('is a step of its own to the palette, every time', () => {
    const { commands } = asking()
    const steps = [commands.step.value]
    commands.asks('child', front())
    steps.push(commands.step.value)
    commands.leaves()
    steps.push(commands.step.value)
    commands.asks('title', front())
    steps.push(commands.step.value)

    expect(new Set(steps).size).toBe(3)
    expect(steps[1]).not.toBe(steps[3])
  })
})

describe('a command asked for from outside the palette', () => {
  it('opens the palette where it asks for what it needs', () => {
    const { commands } = asking()
    commands.shows(false)

    expect(commands.asks('remove', front())).toBeNull()
    expect(commands.open.value).toBe(true)
    expect(commands.crumb.value).toBe(words.remove)
  })

  it('leaves the palette shut where it needs nothing', () => {
    const { commands } = asking()
    commands.shows(false)

    expect(commands.asks('copy', front())?.id).toBe('copy')
    expect(commands.open.value).toBe(false)
  })
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
  const bands = (items: number, working = false) => [
    { id: 'names', title: 'Names', items: Array.from({ length: items }, (_, at) => ({ id: `${at}`, title: 'One' })), working },
  ]

  it('offers to make the note that was looked for', () => {
    const offered = offering(bands(0), 'Entropy', words)

    expect(offered.at(-1)?.id).toBe(MAKING)
    expect(offered.at(-1)?.items[0]?.title).toBe('Create a note called “Entropy”')
  })

  it('offers nothing while a band is still waiting on the vault', () => {
    expect(offering(bands(0, true), 'Entropy', words)).toHaveLength(1)
  })

  it('offers nothing where a band turned something up', () => {
    expect(offering(bands(1), 'Entropy', words)).toHaveLength(1)
  })

  it('offers nothing where nothing was looked for', () => {
    expect(offering(bands(0), '   ', words)).toHaveLength(1)
  })

  it('makes the note where the person is standing, under the words looked for', () => {
    expect(creates('Entropy', front())).toStrictEqual({
      id: 'note',
      path: '',
      title: '',
      name: 'Entropy',
      kind: 'note',
      tab: 'tab',
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
