/**
 * What a step asks for and what it draws while it stands open, without a
 * window.
 *
 * Two things are asked here: that a tab holding something that is not a note
 * is offered nothing to do to a note, and that destroying is reached by typing
 * the name of the note and by nothing else.
 */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { runSupport, type RunSupport } from './runs'
import type { CommandTarget } from './target'
import type { NoteLookup, PaletteLists, StepGroup } from './lists'
import { useCommandPalette } from './palette'
import type { VaultList, Vault } from '../../shared/vaults'
import type { NameMatch } from './search'
import { WORDS as words } from '../../shared/words'

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

/** One name as the vault answers one. */
const name = (path: string, title: string, heading = ''): NameMatch => ({
  path,
  title,
  heading,
  line: heading ? 4 : -1,
  at: [{ from: 0, to: 1 }],
  type: 'note',
})

/** One vault as the list answers one. */
const vault = (id: string, name: string, missing = false): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  missing,
})

/** The vaults the installation holds, with the one in front named. */
const installation = (...vaults: readonly Vault[]): VaultList => ({
  vaults,
  showing: 'physics',
})

/**
 * What the window calls the notes it holds, and which of them a tab holds. A
 * test moves a note under an open step by writing here.
 */
const held = () => {
  const titles = ref<Record<string, string>>({ 'physics/Ontology.md': 'Ontology' })
  const tabs = ref<Record<string, string>>({})
  const knows: NoteLookup = {
    called: (path) => titles.value[path] ?? '',
    holding: (path) => tabs.value[path] ?? null,
  }
  return {
    knows,
    /** The note filed at a name is now filed at another, under another name. */
    moves: (from: string, to: string, title: string) => {
      titles.value = { ...titles.value, [to]: title }
      delete titles.value[from]
      const was = tabs.value[from]
      if (was) tabs.value = { [to]: was }
    },
    /** A tab of the window holds the note filed at a name. */
    opens: (path: string, tab: string) => {
      tabs.value = { ...tabs.value, [path]: tab }
    },
  }
}

/** The lists the window holds, and every row a step said it was standing on. */
const holding = (offers: Record<string, readonly StepGroup[]>) => {
  const lists = ref(offers)
  const shown: string[] = []
  const holds: PaletteLists = {
    offers: (command) => lists.value[command] ?? [],
    shows: (command, item) => void shown.push(`${command} ${item}`),
  }
  return { holds, lists, shown }
}

/** The commands over what a test says is in front, asked without a hold. */
const asking = (
  over: Partial<CommandTarget> = {},
  found: readonly NameMatch[] = [],
  offers: Record<string, readonly StepGroup[]> = {},
  listed: VaultList = installation(vault('physics', 'Physics')),
  runs: RunSupport = runSupport(),
) => {
  const at = ref(front(over))
  const asked: string[] = []
  const core = {
    names: async (query: string) => {
      asked.push(query)
      return found
    },
    vaults: async () => listed,
  }
  const window = held()
  const kept = holding(offers)
  const commands = useCommandPalette(
    core,
    words,
    () => at.value,
    window.knows,
    kept.holds,
    runs,
    async () => {},
  )
  commands.shows(true)
  return { commands, at, asked, ...window, ...kept }
}

/** A moment for the list of vaults to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

/** Every item drawn, group by group, under the group it stands in. */
const drawn = (groups: ReturnType<typeof asking>['commands']['groups']) =>
  Object.fromEntries(groups.value.map((group) => [group.id, group.items.map((item) => item.id)]))

/** What one group says while it holds nothing. */
const silence = (groups: ReturnType<typeof asking>['commands']['groups'], id: string) =>
  groups.value.find((group) => group.id === id)?.silence

describe('the commands as they open', () => {
  it('offers everything that can be done to what is in front, with nothing typed', () => {
    const { commands } = asking()

    expect(drawn(commands.groups)).toStrictEqual({
      note: [
        'read',
        'travel',
        'child',
        'parent',
        'jump',
        'title',
        'remove',
        'ask',
        'copy',
        'reveal',
        'preset',
      ],
      window: [
        'note',
        'deck',
        'stencil',
        'newPreset',
        'importUrl',
        'plex',
        'files',
        'agent',
        'close',
        'find',
        'appearance',
        'mode',
        'interfaceScale',
        'textScale',
        'syncing',
        'hanging',
        'parts',
        'settings',
      ],
      vault: [
        'first',
        'goto',
        'openVault',
        'newVault',
        'renameVault',
        'forgetVault',
        'eraseVault',
      ],
    })
  })

  it('draws the command reached by Shift and Enter on the row that names it', () => {
    const { commands } = asking()
    const items = commands.groups.value[0]?.items ?? []

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
    const window = commands.groups.value[1]?.items ?? []

    expect(window.find((item) => item.id === 'find')?.keys).toBe(words.findKeys)
  })

  it('writes nothing under a command that its whole group is over', () => {
    const { commands } = asking()

    for (const item of commands.groups.value[0]?.items ?? []) expect(item.detail).toBeUndefined()
  })

  it('keeps the commands the words typed leave, and lights where they stand', () => {
    const { commands } = asking()

    void commands.typing('child')

    expect(drawn(commands.groups)).toStrictEqual({ note: ['child'], window: [], vault: [] })
    expect(commands.groups.value[0]?.items[0]?.at).toStrictEqual([{ from: 4, to: 9 }])
  })
})

describe('what is in front', () => {
  it('offers nothing over a note when a document is in front', () => {
    const { commands } = asking({ kind: 'document', path: '', title: '' })

    expect(drawn(commands.groups).note).toStrictEqual([])
    expect(silence(commands.groups, 'note')).toBe(words.noNote)
  })

  it('offers nothing over a note when the tab in front is of no kind', () => {
    const { commands } = asking({ kind: null, path: '', title: '' })

    expect(drawn(commands.groups).note).toStrictEqual([])
  })

  it('offers nothing over a note when the plex is standing nowhere', () => {
    const { commands } = asking({ kind: 'plex', path: '', title: '' })

    expect(drawn(commands.groups).note).toStrictEqual([])
  })

  /** A vault that will not open is the one a person most needs to leave. */
  it('says the vault could not be opened, and offers nothing over the note', () => {
    const { commands } = asking({ ready: false })

    expect(drawn(commands.groups)).toStrictEqual({
      note: [],
      window: [
        'plex',
        'files',
        'agent',
        'close',
        'find',
        'appearance',
        'mode',
        'interfaceScale',
        'textScale',
        'syncing',
        'hanging',
        'parts',
        'settings',
      ],
      vault: ['openVault', 'newVault', 'renameVault', 'forgetVault', 'eraseVault'],
    })
    expect(silence(commands.groups, 'note')).toBe(words.noVault)
  })

  it('offers no close where the window holds no tab at all', () => {
    const { commands } = asking({ tab: '' })

    expect(drawn(commands.groups).window).not.toContain('close')
  })

  it('follows what the person is looking at while it is open', () => {
    const { commands, at } = asking()

    at.value = front({ kind: 'document', path: '', title: '' })

    expect(drawn(commands.groups).note).toStrictEqual([])
  })
})

describe('a command that needs nothing', () => {
  it('is carried out the moment it is chosen, over the note in front', () => {
    const { commands } = asking()

    expect(commands.chose('read', 'read')).toStrictEqual({
      id: 'read',
      path: 'physics/Ontology.md',
      vault: { id: 'physics', name: 'Physics' },
      note: null,
      title: 'Ontology',
      file: '',
      others: [],
      name: '',
      kind: 'note',
      tab: 'tab',
    })
  })

  it('carries the identity of the tab holding the note it is over', () => {
    const { commands, opens } = asking()
    opens('physics/Ontology.md', 'held')

    expect(commands.chose('read', 'read')?.note).toBe('held')
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
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['naming'])
  })

  it('asks for one before a preset is made, as it does before a deck', () => {
    const { commands } = asking()

    expect(commands.asks('newPreset', front())).toBeNull()
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['naming'])
  })

  it('carries the name a preset was asked for under', () => {
    const { commands } = asking()
    commands.asks('newPreset', front())
    void commands.typing('Sanskrit')

    expect(commands.chose('name', 'name')?.name).toBe('Sanskrit')
  })

  it('offers what was typed as the name, and nothing before anything is', () => {
    const { commands } = asking()
    commands.asks('child', front())

    expect(commands.groups.value[0]?.items).toStrictEqual([])

    void commands.typing('  Entropy  ')

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Call it “Entropy”')
  })

  it('carries the name to the note it was asked over', () => {
    const { commands } = asking()
    commands.asks('child', front())
    void commands.typing('Entropy')

    expect(commands.chose('name', 'name')).toStrictEqual({
      id: 'child',
      path: 'physics/Ontology.md',
      vault: { id: 'physics', name: 'Physics' },
      note: null,
      title: 'Ontology',
      file: '',
      others: [],
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

describe('a command that asks for an address', () => {
  it('stands on a step of its own', () => {
    const { commands } = asking()

    expect(commands.asks('importUrl', front())).toBeNull()
    expect(commands.crumb.value).toBe(words.importUrl)
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['address'])
  })

  it('offers what was typed where a browser would go there', () => {
    const { commands } = asking()
    commands.asks('importUrl', front())

    expect(commands.groups.value[0]?.items).toStrictEqual([])

    void commands.typing('  https://youtu.be/dQw4w9WgXcQ  ')

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Import “https://youtu.be/dQw4w9WgXcQ”')
  })

  it('offers nothing for words no browser would go to, and says so', () => {
    const { commands } = asking()
    commands.asks('importUrl', front())

    for (const typed of ['Entropy', 'file:///etc/passwd', 'example.com/a', 'https://']) {
      void commands.typing(typed)

      expect(commands.groups.value[0]?.items, typed).toStrictEqual([])
      expect(commands.groups.value[0]?.silence, typed).toBe(words.notAnAddress)
      expect(commands.chose('name', 'name'), typed).toBeNull()
    }
  })

  it('carries the address to whatever makes the note', () => {
    const { commands } = asking()
    commands.asks('importUrl', front())
    void commands.typing('https://example.com/a')

    expect(commands.chose('name', 'name')?.name).toBe('https://example.com/a')
  })
})

/** The note goes to the vault's .trash folder, so nothing is asked over it. */
describe('removing a note', () => {
  it('is carried out the moment it is asked for, over the note in front', () => {
    const { commands } = asking()

    const invocation = commands.asks('remove', front())

    expect(invocation?.id).toBe('remove')
    expect(invocation?.path).toBe('physics/Ontology.md')
  })

  it('carries every file it was over into the invocation', () => {
    const { commands } = asking()
    const others = ['physics/Heat.pdf']

    expect(commands.asks('remove', front({ others }))?.others).toStrictEqual(others)
  })

  it('carries the tab holding it, so the invocation reaches it wherever it went', () => {
    const { commands, opens } = asking()
    opens('physics/Ontology.md', 'held')

    expect(commands.asks('remove', front())?.note).toBe('held')
  })

  it('leaves the palette on the commands, having nothing to ask', () => {
    const { commands } = asking()

    commands.asks('remove', front())

    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['note', 'window', 'vault'])
  })
})

describe('destroying a note', () => {
  it('waits for the name of the note, and is not reachable before it', () => {
    const { commands } = asking()
    commands.asks('destroy', front())

    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(true)
    expect(commands.chose('exactly', 'exactly')).toBeNull()

    void commands.typing('Ontolog')

    expect(commands.chose('exactly', 'exactly')).toBeNull()
  })

  it('goes once the name of the note is what was typed', () => {
    const { commands } = asking()
    commands.asks('destroy', front())
    void commands.typing('Ontology')

    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(false)
    expect(commands.chose('exactly', 'exactly')?.id).toBe('destroy')
  })
})

/**
 * The vault moves while a person is answering. A step is about the note they
 * named, and the name that note is filed under is not what it was.
 */
describe('a note that moves under an open step', () => {
  it('is renamed where it now is, and never at the name it left', () => {
    const { commands, moves } = asking()
    commands.asks('title', front())
    moves('physics/Ontology.md', 'physics/Being.md', 'Being')
    commands.follows([{ from: 'physics/Ontology.md', to: 'physics/Being.md' }])

    void commands.typing('Substance')

    expect(commands.chose('name', 'name')?.path).toBe('physics/Being.md')
  })

  it('is not destroyed by the name it had, and is by the name it has', () => {
    const { commands, moves } = asking()
    commands.asks('destroy', front())
    moves('physics/Ontology.md', 'physics/Being.md', 'Being')
    commands.follows([{ from: 'physics/Ontology.md', to: 'physics/Being.md' }])

    void commands.typing('Ontology')

    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(true)
    expect(commands.chose('exactly', 'exactly')).toBeNull()

    void commands.typing('Being')

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Destroy “Being”')
    expect(commands.chose('exactly', 'exactly')?.path).toBe('physics/Being.md')
  })

  it('carries the tab holding it, so the invocation reaches it wherever it went', () => {
    const { commands, opens } = asking()
    opens('physics/Ontology.md', 'held')
    commands.asks('title', front())

    void commands.typing('Substance')

    expect(commands.chose('name', 'name')?.note).toBe('held')
  })

  it('leaves a step over another note where it stands', () => {
    const { commands } = asking()
    commands.asks('title', front())

    commands.follows([{ from: 'Elsewhere.md', to: 'Moved.md' }])

    void commands.typing('Substance')

    expect(commands.chose('name', 'name')?.path).toBe('physics/Ontology.md')
  })
})

/** A recording in front of the window, and a scanned document. */
const recording: Partial<CommandTarget> = {
  kind: 'recording',
  path: '',
  // A recording tab is filed at no note, and what it is called is the name of
  // the file it plays.
  title: 'Ants.mp3',
  file: 'talks/Ants.mp3',
  source: 'recording',
}
const scanned: Partial<CommandTarget> = {
  kind: 'document',
  path: '',
  title: '',
  file: 'books/Ants.pdf',
  source: 'book',
}

describe('the runs over the file in front', () => {
  it('offers a recording to be transcribed, put right and dropped, and nothing to recognise', () => {
    const { commands } = asking(recording)

    expect(drawn(commands.groups).file).toStrictEqual([
      'transcribe',
      'proofread',
      'deleteText',
    ])
  })

  it('offers a scan to be recognised, and nothing to transcribe', () => {
    const { commands } = asking(scanned)

    expect(drawn(commands.groups).file).toStrictEqual(['recognise'])
  })

  // The group stands where there is something in it, so a note is offered no
  // empty shelf of runs.
  it('offers neither over a note, and draws no group for them', () => {
    const { commands } = asking()

    expect(drawn(commands.groups).file).toBeUndefined()
  })

  it('offers neither where the vault could not be opened', () => {
    const { commands } = asking({ ...recording, ready: false })

    expect(drawn(commands.groups).file).toBeUndefined()
  })

  it('carries the file the tab in front holds', () => {
    const { commands } = asking(recording)

    expect(commands.chose('transcribe', 'transcribe')?.file).toBe('talks/Ants.mp3')
  })

  it('is offered nowhere once this build has said it cannot do it at all', () => {
    const runs = runSupport()
    runs.cannotRun('transcribe')
    runs.cannotRun('proofread')
    runs.cannotRun('deleteText')

    expect(drawn(asking(recording, [], {}, undefined, runs).commands.groups).file).toBeUndefined()
    expect(drawn(asking(scanned, [], {}, undefined, runs).commands.groups).file).toStrictEqual([
      'recognise',
    ])
  })

  // A run is offered on what has been made from the file and not on its kind.
  // The window can ask what a book carries, so a book already read is not
  // offered to be read again.
  it('offers a scan to be recognised only where nothing has read it', () => {
    for (const [made, offered] of [
      ['none', true],
      ['stopped', true],
      ['queued', false],
      ['running', false],
      ['done', false],
      ['empty', false],
      ['failed', false],
    ] as const) {
      const { commands } = asking({ ...scanned, made: { ocr: made } })

      expect(drawn(commands.groups).file, made).toStrictEqual(offered ? ['recognise'] : undefined)
    }
  })

  it('offers a recording to be transcribed only where nothing has transcribed it', () => {
    const { commands } = asking({ ...recording, made: { transcript: 'done' } })

    expect(drawn(commands.groups).file).not.toContain('transcribe')
  })

  // There is nothing to put right until a model has transcribed something, and
  // nothing to take away until it has.
  it('offers a transcript to be put right once one stands, and not before', () => {
    expect(drawn(asking({ ...recording, made: { transcript: 'none' } }).commands.groups).file)
      .toStrictEqual(['transcribe'])
    expect(drawn(asking({ ...recording, made: { transcript: 'done' } }).commands.groups).file)
      .toStrictEqual(['proofread', 'deleteText'])
    expect(
      drawn(asking({ ...recording, made: { transcript: 'done', 'transcript.corrected': 'done' } }).commands.groups)
        .file,
    ).toStrictEqual(['deleteText'])
  })

  // Nothing has been asked yet, and the file is offered what its kind offers.
  // A window that hid the runs until the answer came would flicker every one of
  // them into place.
  it('offers what the kind offers while nothing is known of the file', () => {
    const { commands } = asking(recording)

    expect(drawn(commands.groups).file).toStrictEqual([
      'transcribe',
      'proofread',
      'deleteText',
    ])
  })

  it('is still offered in a window that has not been told it', () => {
    const runs = runSupport()
    runs.cannotRun('transcribe')
    runs.cannotRun('proofread')
    runs.cannotRun('deleteText')

    expect(drawn(asking(recording, [], {}, undefined, runs).commands.groups).file).toBeUndefined()
    expect(drawn(asking(recording).commands.groups).file).toStrictEqual([
      'transcribe',
      'proofread',
      'deleteText',
    ])
  })
})

describe('dropping the transcript of a recording', () => {
  it('asks before the words go, and is nothing until the answer is given', () => {
    const { commands } = asking(recording)

    expect(commands.asks('deleteText', front(recording))).toBeNull()
    expect(commands.groups.value[0]?.id).toBe('asking')
    expect(commands.groups.value[0]?.items.map((one) => one.id)).toStrictEqual(['no', 'yes'])
  })

  // The answer that changes nothing is the one the keyboard opens on.
  it('names the recording in the answer that takes the words away', () => {
    const { commands } = asking(recording)
    commands.asks('deleteText', front(recording))

    expect(commands.groups.value[0]?.items[0]?.title).toBe(words.keepsTranscript)
    expect(commands.groups.value[0]?.items[1]?.title).toBe(`${words.deletes} “${recording.title}”`)
    expect(commands.groups.value[0]?.items[1]?.detail).toBe(words.deleted)
  })

  it('carries the recording the tab in front holds once the answer is given', () => {
    const { commands } = asking(recording)
    commands.asks('deleteText', front(recording))

    expect(commands.chose('yes', 'yes')?.file).toBe('talks/Ants.mp3')
  })

  it('does nothing and puts the step away where the answer keeps the words', () => {
    const { commands } = asking(recording)
    commands.asks('deleteText', front(recording))

    expect(commands.chose('no', 'no')).toBeNull()
    expect(commands.groups.value[0]?.id).not.toBe('asking')
  })
})

describe('a command that was not offered over what it was asked over', () => {
  it('says the vault could not be opened', () => {
    const { commands } = asking({ ready: false })

    expect(commands.refused('remove', front({ ready: false }))).toBe(words.noVault)
  })

  it('says nothing in front is a note', () => {
    const { commands } = asking()

    expect(commands.refused('remove', front({ kind: 'document', path: '', title: '' }))).toBe(
      words.noNote,
    )
  })

  it('says nothing at all about one that was taken up', () => {
    const { commands } = asking()

    expect(commands.refused('remove', front())).toBe('')
    expect(commands.refused('nothing of the sort', front())).toBe('')
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
    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
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
      vault: { id: 'physics', name: 'Physics' },
      note: null,
      title: 'Entropy',
      file: '',
      others: [],
      name: '',
      kind: 'note',
      tab: 'tab',
    })
  })

  it('says the vault could not answer, in the window’s own words', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    const at = ref(front())
    const commands = useCommandPalette(
      {
        names: async () => Promise.reject(new Error('no model is set')),
        vaults: async () => installation(),
      },
      words,
      () => at.value,
      { called: () => '', holding: () => null },
      { offers: () => [], shows: () => {} },
      runSupport(),
      async () => {},
    )
    commands.shows(true)
    commands.asks('goto', front())

    await commands.typing('en')

    expect(commands.groups.value[0]?.silence).toBe(words.notAsked)
  })
})

describe('a command that offers a list the window holds', () => {
  /** The themes, in the two groups they come off, and one group holding none. */
  const THEMES: readonly StepGroup[] = [
    {
      id: 'shipping',
      title: 'Ships with numen',
      items: [
        { id: 'preset:numen', title: 'numen', detail: 'Current' },
        { id: 'preset:dracula', title: 'dracula' },
      ],
    },
    { id: 'owned', title: 'Your own themes', items: [], silence: 'A .css file there is one' },
  ]

  /** The three halves, drawn and not to be chosen, as a pinned theme leaves them. */
  const HALVES: readonly StepGroup[] = [
    {
      id: 'half',
      title: 'Light and dark',
      items: [
        { id: 'mode:system', title: 'Follow the system', disabled: true },
        { id: 'mode:light', title: 'Light', detail: 'The theme worn sets this itself' },
      ],
    },
  ]

  const held = { appearance: THEMES, mode: HALVES }

  it('offers the groups the window holds, each under its own name', () => {
    const { commands } = asking({}, [], held)
    commands.asks('appearance', front())

    expect(commands.crumb.value).toBe(words.appearance)
    expect(commands.placeholder.value).toBe(words.typeChoice)
    expect(commands.groups.value.map((group) => group.title)).toStrictEqual([
      'Ships with numen',
      'Your own themes',
    ])
    expect(drawn(commands.groups)).toStrictEqual({
      shipping: ['preset:numen', 'preset:dracula'],
      owned: [],
    })
    expect(silence(commands.groups, 'owned')).toBe('A .css file there is one')
  })

  it('offers each command the list it holds for that command', () => {
    const { commands } = asking({}, [], held)
    commands.asks('mode', front())

    expect(commands.crumb.value).toBe(words.mode)
    expect(drawn(commands.groups)).toStrictEqual({
      half: ['mode:system', 'mode:light'],
    })
  })

  it('keeps to the rows the words typed name, and marks where they stand', async () => {
    const { commands } = asking({}, [], held)
    commands.asks('appearance', front())

    await commands.typing('rac')

    expect(drawn(commands.groups)).toStrictEqual({ shipping: ['preset:dracula'], owned: [] })
    expect(commands.groups.value[0]?.items[0]?.at).toStrictEqual([{ from: 1, to: 4 }])
  })

  it('draws a row the window says cannot be chosen, and chooses nothing by it', () => {
    const { commands } = asking({}, [], held)
    commands.asks('mode', front())

    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(true)
    expect(commands.chose('mode:system', 'chosen')).toBeNull()
    expect(commands.chose('mode:none', 'chosen')).toBeNull()
  })

  it('hands back the row that was chosen, from whichever group it stood in', () => {
    const { commands } = asking({}, [], held)
    commands.asks('appearance', front())

    expect(commands.chose('preset:dracula', 'chosen')).toStrictEqual({
      id: 'appearance',
      path: 'physics/Ontology.md',
      note: null,
      title: 'Ontology',
      file: '',
      others: [],
      name: 'preset:dracula',
      kind: 'note',
      tab: 'tab',
      vault: { id: 'physics', name: 'Physics' },
    })
  })

  it('says which command the keyboard is standing in, and that it stands nowhere after', () => {
    const { commands, shown } = asking({}, [], held)
    commands.asks('mode', front())

    commands.lights('mode:light')
    commands.leaves()
    commands.asks('appearance', front())
    commands.lights('preset:dracula')
    commands.leaves()

    expect(shown).toStrictEqual([
      'mode mode:light',
      'mode ',
      'appearance preset:dracula',
      'appearance ',
    ])
  })

  it('says nothing about where the keyboard is at a step that shows nothing', () => {
    const { commands, shown } = asking({}, [], held)
    commands.asks('child', front())

    commands.lights('anything')
    commands.leaves()

    expect(shown).toStrictEqual([])
  })
})

describe('leaving a step', () => {
  it('goes back to the commands, and the palette stays open', () => {
    const { commands } = asking()
    commands.asks('child', front())

    commands.leaves()

    expect(commands.open.value).toBe(true)
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['note', 'window', 'vault'])
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

    expect(commands.asks('title', front())).toBeNull()
    expect(commands.open.value).toBe(true)
    expect(commands.crumb.value).toBe(words.title)
  })

  it('leaves the palette shut where it needs nothing', () => {
    const { commands } = asking()
    commands.shows(false)

    expect(commands.asks('copy', front())?.id).toBe('copy')
    expect(commands.open.value).toBe(false)
  })
})

describe('the commands over the vaults an installation holds', () => {
  /** Only renaming is over the vault in front. The rest ask for one first. */
  it('leave out the rename where the window is showing no vault', () => {
    const { commands } = asking({ vault: { id: '', name: '' } })

    expect(drawn(commands.groups).vault).toStrictEqual([
      'first',
      'goto',
      'openVault',
      'newVault',
      'forgetVault',
      'eraseVault',
    ])
  })
})

describe('a command that asks for a vault', () => {
  const two = () => installation(vault('physics', 'Physics'), vault('heat', 'Heat'))

  it('lists the vaults the installation holds, each at the folder it stands in', async () => {
    const { commands } = asking({}, [], {}, two())
    commands.asks('openVault', front())

    await settles()

    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual(['physics', 'heat'])
    expect(commands.groups.value[0]?.items[1]?.detail).toBe('/vaults/Heat')
  })

  /** The folder may be on a volume nobody has mounted, and the vault stays. */
  it('marks the vault whose folder is not there, and names where it looked', async () => {
    const { commands } = asking({}, [], {}, installation(vault('gone', 'Gone', true)))
    commands.asks('openVault', front())
    await settles()

    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(true)
    expect(commands.groups.value[0]?.items[0]?.detail).toBe(`${words.gone} · /vaults/Gone`)
    expect(commands.chose('gone', 'open')).toBeNull()
  })

  /**
   * The window always stands on something, so the vault in front is not one
   * the list takes. It is drawn with the reason beside it, and choosing it
   * does nothing.
   */
  it('marks the vault the window is showing, and will not take it', async () => {
    const { commands } = asking({}, [], {}, two())
    commands.asks('openVault', front())
    await settles()

    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(true)
    expect(commands.groups.value[0]?.items[0]?.detail).toBe(`${words.current} · /vaults/Physics`)
    expect(commands.chose('physics', 'open')).toBeNull()
  })

  it('says on every row what choosing it asks for', async () => {
    const { commands } = asking({}, [], {}, two())
    commands.asks('forgetVault', front())
    await settles()

    expect(commands.groups.value[0]?.items[1]?.actions?.[0]?.text).toBe(words.forgetVault)
  })

  it('carries the vault that was picked, not the one the window is showing', async () => {
    const { commands } = asking({}, [], {}, two())
    commands.asks('openVault', front())
    await settles()

    expect(commands.chose('heat', 'open')).toMatchObject({
      id: 'openVault',
      vault: { id: 'heat', name: 'Heat' },
    })
  })

  it('keeps the vaults the words typed name, so the field means something', async () => {
    const { commands } = asking({}, [], {}, two())
    commands.asks('openVault', front())
    await settles()

    await commands.typing('hea')

    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual(['heat'])
  })

  it('says the list could not be asked, in the window’s own words', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    const at = ref(front())
    const commands = useCommandPalette(
      {
        names: async () => [],
        vaults: async () => Promise.reject(new Error('the list is not there')),
      },
      words,
      () => at.value,
      { called: () => '', holding: () => null },
      { offers: () => [], shows: () => {} },
      runSupport(),
      async () => {},
    )
    commands.shows(true)

    commands.asks('openVault', front())
    await settles()

    expect(commands.groups.value[0]?.silence).toBe(words.notAsked)
  })
})

describe('renaming the vault in front', () => {
  it('stands filled with the name the vault has now', () => {
    const { commands } = asking()

    commands.asks('renameVault', front())

    expect(commands.crumb.value).toBe(words.renameVault)
    expect(commands.typed.value).toBe('Physics')
  })

  it('carries the name to the vault the window is showing', () => {
    const { commands } = asking()
    commands.asks('renameVault', front())
    void commands.typing('Heat')

    expect(commands.chose('name', 'name')).toMatchObject({
      id: 'renameVault',
      name: 'Heat',
      vault: { id: 'physics', name: 'Physics' },
    })
  })
})

/**
 * The window always stands on a vault, so neither of these is over the one in
 * front. Both ask for a vault first, and everything they say after is about
 * the vault that was chosen.
 */
describe('a vault taken off the list', () => {
  /** The list opened, and the vault the window is not showing chosen off it. */
  const chose = async (id: string) => {
    const { commands } = asking(
      {},
      [],
      {},
      installation(vault('physics', 'Physics'), vault('heat', 'Heat')),
    )
    commands.asks(id, front())
    await settles()
    commands.chose('heat', 'open')
    return commands
  }

  it('is confirmed over the vault that was chosen, saying the folder stays', async () => {
    const commands = await chose('forgetVault')

    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual(['no', 'yes'])
    expect(commands.groups.value[0]?.items[0]?.title).toBe(words.keepsVault)
    expect(commands.groups.value[0]?.items[1]?.title).toBe('Forget “Heat”')
    expect(commands.groups.value[0]?.items[1]?.detail).toBe(words.stays)
  })

  it('is carried out over that vault once the question is answered', async () => {
    const commands = await chose('forgetVault')

    expect(commands.chose('yes', 'yes')).toMatchObject({
      id: 'forgetVault',
      vault: { id: 'heat', name: 'Heat' },
    })
  })

  it('hands back the list, with the vaults on it, where the step is left', async () => {
    const commands = await chose('forgetVault')

    commands.leaves()
    await settles()

    expect(commands.groups.value[0]?.id).toBe('vaults')
    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics',
      'heat',
    ])
  })

  it('waits for the name of the vault chosen, and says where its folder goes', async () => {
    const commands = await chose('eraseVault')

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Erase “Heat”')
    expect(commands.groups.value[0]?.items[0]?.detail).toBe(words.binned)
    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(true)
    expect(commands.placeholder.value).toBe(words.typeVaultBack)
    expect(commands.chose('exactly', 'exactly')).toBeNull()
  })

  it('goes once the name of that vault is what was typed', async () => {
    const commands = await chose('eraseVault')

    void commands.typing('Heat')

    expect(commands.groups.value[0]?.items[0]?.disabled).toBe(false)
    expect(commands.chose('exactly', 'exactly')).toMatchObject({
      id: 'eraseVault',
      vault: { id: 'heat', name: 'Heat' },
    })
  })

  it('is not reached by the name of the vault the window is showing', async () => {
    const commands = await chose('eraseVault')

    void commands.typing('Physics')

    expect(commands.chose('exactly', 'exactly')).toBeNull()
  })
})
