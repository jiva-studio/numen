/**
 * Palette step requests and the active step's rendering, without a window.
 *
 * Covered: a tab holding a non-note is offered no note actions, and a
 * destructive action is reached only by typing the note's name.
 */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { runSupport } from './runSupport'
import type { StepGroup } from '@/shared/model/command'
import type { NoteLookup, PaletteLists } from '../../types'
import type { Command } from '@/entities/command'
import type { CommandTarget, RunSupport } from '@/shared/model/command'
import { createCommandPalette, type Commands } from './palette'
import type { VaultList, Vault } from '@/entities/vault'
import type { NameMatch } from '../search/search'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../../words'
import { createAppCommands } from '@/testing/appCommands'
import { createDeferred, type Deferred } from '@/testing/deferred'

const words = { ...windowWords, ...PALETTE_WORDS }

function rethrow(error: unknown): never {
  throw error
}

function chooseRow(commands: Commands, which: number | string = 0) {
  const items = commands.view.groups.value.flatMap((group) => group.items)
  const row = typeof which === 'number' ? items[which] : items.find((one) => one.id === which)
  return commands.palette.chooseItem(row?.id ?? String(which), row?.actions?.[0]?.id ?? '')
}

const front = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...override,
})

const name = (
  path: string,
  title: string,
  heading = '',
  type: NameMatch['type'] = 'note',
): NameMatch => ({
  path,
  title,
  heading,
  line: heading ? 4 : -1,
  at: [{ start: 0, end: 1 }],
  type,
  ...(type === 'note' ? {} : { tabKind: type }),
})

const vault = (id: string, name: string, isMissing = false): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: isMissing,
})

const installation = (...vaults: readonly Vault[]): VaultList => ({
  vaults,
  showing: 'physics',
})

const createLookup = () => {
  const titles = ref<Record<string, string>>({ 'physics/Ontology.md': 'Ontology' })
  const tabs = ref<Record<string, string>>({})
  const knows: NoteLookup = {
    getTitle: (path) => titles.value[path] ?? '',
    getTabAt: (path) => tabs.value[path] ?? null,
  }
  return {
    knows,
    /** The note filed at a name is now filed at another, under another name. */
    moveNote: (from: string, to: string, title: string) => {
      titles.value = { ...titles.value, [to]: title }
      delete titles.value[from]
      const was = tabs.value[from]
      if (was) tabs.value = { [to]: was }
    },
    /** A tab of the window holds the note filed at a name. */
    openNote: (path: string, tab: string) => {
      tabs.value = { ...tabs.value, [path]: tab }
    },
  }
}

const THEME_ID = 'setting:app:appearance.theme'
const MODE_ID = 'setting:app:appearance.mode'
const THEME_TEXT = 'Theme'
const MODE_TEXT = 'Light or dark'

/** The commands of two settings, as the adapter of the settings offers them beside the table. */
const SETTING_COMMANDS: readonly Command[] = [
  { id: THEME_ID, text: THEME_TEXT, group: 'settings', input: 'items', isAvailable: () => true },
  { id: MODE_ID, text: MODE_TEXT, group: 'settings', input: 'items', isAvailable: () => true },
]

const createLists = (commandLists: Record<string, readonly StepGroup[]>) => {
  const lists = ref(commandLists)
  const displayedItems: string[] = []
  const paletteLists: PaletteLists = {
    getStepGroups: (command) => lists.value[command] ?? [],
    previewItem: (command, item) => {
      displayedItems.push(`${command} ${item}`)
    },
  }
  return { paletteLists, lists, displayedLists: displayedItems }
}

const createPalette = (
  override: Partial<CommandTarget> = {},
  found: readonly NameMatch[] = [],
  entries: Record<string, readonly StepGroup[]> = {},
  items: VaultList = installation(vault('physics', 'Physics')),
  runs: RunSupport = runSupport(),
) => {
  const activeTarget = ref(front(override))
  const request: string[] = []
  const core = {
    names: async (query: string) => {
      request.push(query)
      return found
    },
    vaults: async () => items,
  }
  const window = createLookup()
  const retainedItem = createLists(entries)
  const commands = createCommandPalette({
    core,
    words,
    at: () => activeTarget.value,
    knows: window.knows,
    lists: retainedItem.paletteLists,
    runs,
    wait: vi.fn(() => Promise.resolve()),
    getCommands: () => [
      ...createAppCommands(),
      ...(Object.keys(entries).length > 0 ? SETTING_COMMANDS : []),
    ],
  })
  commands.palette.setOpen(true)
  return {
    commands,
    at: activeTarget,
    target: activeTarget,
    requests: request,
    ...window,
    ...retainedItem,
  }
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const getItemIds = (groups: ReturnType<typeof createPalette>['commands']['view']['groups']) =>
  Object.fromEntries(groups.value.map((group) => [group.id, group.items.map((item) => item.id)]))

const getEmptyMessage = (
  groups: ReturnType<typeof createPalette>['commands']['view']['groups'],
  id: string,
) => groups.value.find((group) => group.id === id)?.silence

describe('the commands as they open', () => {
  it('offers everything that can be done to what is in front, with nothing typed', () => {
    const { commands } = createPalette()

    expect(getItemIds(commands.view.groups)).toStrictEqual({
      file: ['app:version.delete'],
      note: [
        'app:file.open',
        'app:note.createChild',
        'app:note.createParent',
        'app:note.createJump',
        'app:note.rename',
        'app:file.remove',
        'app:file.copyPath',
      ],
      window: [
        'app:note.create',
        'app:folder.create',
        'app:tab.close',
        'app:vault.search',
        'app:tab.detach',
      ],
      settings: ['app:settings.open'],
      vault: [
        'app:file.open#note',
        'app:vault.open',
        'app:vault.create',
        'app:vault.rename',
        'app:vault.remove',
        'app:vault.erase',
        'app:trash.empty',
      ],
    })
  })

  it('draws the command reached by Shift and Enter on the row that names it', () => {
    const { commands } = createPalette()
    const items = commands.view.groups.value[0]?.items ?? []

    expect(
      items.find((item) => item.id === 'app:file.open')?.actions?.map((one) => one.id),
    ).toStrictEqual(['app:file.open', 'app:file.open#beside'])
    expect(
      items.find((item) => item.id === 'app:file.remove')?.actions?.map((one) => one.id),
    ).toStrictEqual(['app:file.remove'])
  })

  it('says which key reaches a command away from the palette', () => {
    const { commands } = createPalette()
    const window = commands.view.groups.value.find((group) => group.id === 'window')?.items ?? []

    expect(window.find((item) => item.id === 'app:vault.search')?.keys).toBe(words.findKeys)
  })

  it('writes nothing under a command that its whole group is over', () => {
    const { commands } = createPalette()

    for (const item of commands.view.groups.value[0]?.items ?? [])
      expect(item.detail).toBeUndefined()
  })

  it('keeps the commands the words typed leave, and lights where they match', () => {
    const { commands } = createPalette()

    commands.palette.setTyped('child').catch(rethrow)

    expect(getItemIds(commands.view.groups)).toStrictEqual({ note: ['app:note.createChild'] })
    expect(commands.view.groups.value[0]?.items[0]?.at).toStrictEqual([{ start: 7, end: 12 }])
  })
})

describe('what is in front', () => {
  it('offers nothing over a note when a document is in front', () => {
    const { commands } = createPalette({ kind: 'document', path: '', title: '' })

    expect(getItemIds(commands.view.groups).note).toBeUndefined()
  })

  it('offers nothing over a note when the tab in front is of no kind', () => {
    const { commands } = createPalette({ kind: null, path: '', title: '' })

    expect(getItemIds(commands.view.groups).note).toBeUndefined()
  })

  it('offers nothing over a note when the plex is standing nowhere', () => {
    const { commands } = createPalette({ kind: 'plex', path: '', title: '' })

    expect(getItemIds(commands.view.groups).note).toBeUndefined()
  })

  /** A vault that cannot be opened must still be possible to leave. */
  it('says the vault could not be opened, and offers nothing over the note', () => {
    const { commands } = createPalette({ isReady: false })

    expect(getItemIds(commands.view.groups)).toStrictEqual({
      file: ['app:version.delete'],
      window: ['app:tab.close', 'app:vault.search', 'app:tab.detach'],
      settings: ['app:settings.open'],
      vault: [
        'app:vault.open',
        'app:vault.create',
        'app:vault.rename',
        'app:vault.remove',
        'app:vault.erase',
      ],
    })
    expect(commands.view.groups.value.some((g) => g.id === 'note')).toBe(false)
  })

  it('offers no close where the window holds no tab at all', () => {
    const { commands } = createPalette({ tab: '' })

    expect(getItemIds(commands.view.groups).window).not.toContain('app:tab.close')
  })

  it('follows what the person is looking at while it is open', () => {
    const { commands, target: activeTarget } = createPalette()

    activeTarget.value = front({ kind: 'document', path: '', title: '' })

    expect(getItemIds(commands.view.groups).note).toBeUndefined()
  })
})

describe('a command that needs nothing', () => {
  it('is carried out the moment it is chosen, over the note in front', () => {
    const { commands } = createPalette()

    expect(commands.palette.chooseItem('app:file.open', 'app:file.open')).toStrictEqual({
      id: 'app:file.open',
      target: {
        path: 'physics/Ontology.md',
        vault: { id: 'physics', name: 'Physics' },
        title: 'Ontology',
        file: '',
        others: [],
        kind: 'note',
        tab: 'tab',
      },
      note: null,
      name: '',
    })
  })

  it('carries the identity of the tab holding the note it is over', () => {
    const { commands, openNote } = createPalette()
    openNote('physics/Ontology.md', 'active')

    expect(commands.palette.chooseItem('app:file.open', 'app:file.open')?.note).toBe('active')
  })

  it('is the second one where Shift and Enter reached it', () => {
    const { commands } = createPalette()

    expect(commands.palette.chooseItem('app:file.open', 'app:file.open#beside')).toMatchObject({
      id: 'app:file.open',
      values: { placement: 'split' },
    })
  })

  it('is nothing where what is in front does not offer it', () => {
    const { commands } = createPalette({ kind: 'document', path: '', title: '' })

    expect(commands.palette.chooseItem('app:file.open', 'app:file.open')).toBeNull()
  })
})

describe('the command that makes a note, run over a folder', () => {
  const folder = (override: Partial<CommandTarget> = {}) =>
    front({ path: 'physics', file: 'physics', isFolder: true, kind: null, ...override })

  it('starts without a name where a caller named the folder', () => {
    const { commands } = createPalette()

    const started = commands.palette.startCommand(
      'app:note.create',
      folder({ isTargetNamed: true }),
    )

    expect(started).toMatchObject({
      id: 'app:note.create',
      target: {
        path: 'physics',
        isFolder: true,
      },
      name: '',
    })
  })

  it('asks for a name where no caller named the folder', () => {
    const { commands } = createPalette()

    expect(commands.palette.startCommand('app:note.create', folder())).toBeNull()
    expect(commands.view.groups.value.map((group) => group.id)).toStrictEqual(['naming'])
  })

  it('asks for a name over a note a caller named', () => {
    const { commands } = createPalette()

    expect(
      commands.palette.startCommand('app:note.create', front({ isTargetNamed: true })),
    ).toBeNull()
  })

  it('starts the command that makes a folder without a name where a caller named the folder', () => {
    const { commands } = createPalette()

    const started = commands.palette.startCommand(
      'app:folder.create',
      folder({ isTargetNamed: true }),
    )

    expect(started).toMatchObject({
      id: 'app:folder.create',
      target: {
        path: 'physics',
        isFolder: true,
      },
      name: '',
    })
  })

  it('asks for the path of a folder where no caller named the folder', () => {
    const { commands } = createPalette()

    expect(commands.palette.startCommand('app:folder.create', folder())).toBeNull()
    expect(commands.view.groups.value.map((group) => group.id)).toStrictEqual(['naming'])
  })

  it('asks for a name in a command of another kind over a folder a caller named', () => {
    const { commands } = createPalette()

    expect(
      commands.palette.startCommand('app:note.rename', folder({ isTargetNamed: true })),
    ).toBeNull()
  })
})

describe('a command that asks for a name', () => {
  it('relies on a step of its own, and says which step it is', () => {
    const { commands } = createPalette()

    expect(commands.palette.startCommand('app:note.createChild', front())).toBeNull()
    expect(commands.navigation.crumb.value).toBe(words.child)
    expect(commands.view.groups.value.map((group) => group.id)).toStrictEqual(['naming'])
  })

  it('offers what was typed as the name, and nothing before anything is', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.createChild', front())

    expect(commands.view.groups.value[0]?.items).toStrictEqual([])

    commands.palette.setTyped('  Entropy  ').catch(rethrow)

    expect(commands.view.groups.value[0]?.items[0]?.title).toBe('Name it “Entropy”')
  })

  it('carries the name to the note it was requested over', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.createChild', front())
    commands.palette.setTyped('Entropy').catch(rethrow)

    expect(chooseRow(commands)).toStrictEqual({
      id: 'app:note.createChild',
      target: {
        path: 'physics/Ontology.md',
        vault: { id: 'physics', name: 'Physics' },
        title: 'Ontology',
        file: '',
        others: [],
        kind: 'note',
        tab: 'tab',
      },
      note: null,
      name: 'Entropy',
    })
  })

  it('is nothing while nothing has been typed', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.createChild', front())

    expect(chooseRow(commands)).toBeNull()
  })

  it('puts the title the note has now in the field, for a change of title', () => {
    const { commands } = createPalette()

    commands.palette.startCommand('app:note.rename', front())

    expect(commands.palette.typed.value).toBe('Ontology')
  })
})

/** The note moves to the vault's .trash folder. */
describe('removing a note', () => {
  it('is carried out the moment it is requested for, over the note in front', () => {
    const { commands } = createPalette()

    const invocation = commands.palette.startCommand('app:file.remove', front())

    expect(invocation?.id).toBe('app:file.remove')
    expect(invocation?.target.path).toBe('physics/Ontology.md')
  })

  it('carries every file it was over into the invocation', () => {
    const { commands } = createPalette()
    const others = ['physics/Heat.pdf']

    expect(
      commands.palette.startCommand('app:file.remove', front({ others }))?.target.others,
    ).toStrictEqual(others)
  })

  it('carries the tab holding it, so the invocation reaches it wherever it went', () => {
    const { commands, openNote } = createPalette()
    openNote('physics/Ontology.md', 'active')

    expect(commands.palette.startCommand('app:file.remove', front())?.note).toBe('active')
  })

  it('leaves the palette on the commands, having nothing to ask', () => {
    const { commands } = createPalette()

    commands.palette.startCommand('app:file.remove', front())

    expect(commands.view.groups.value.map((group) => group.id)).toStrictEqual([
      'note',
      'file',
      'window',
      'settings',
      'vault',
    ])
  })
})

/**
 * The vault changes while a step is active. A step targets the note the user
 * named, and the note's path differs from the original.
 */
describe('a note that moves under an open step', () => {
  it('is renamed where it now is, and never at the name it left', () => {
    const { commands, moveNote } = createPalette()
    commands.palette.startCommand('app:note.rename', front())
    moveNote('physics/Ontology.md', 'physics/Being.md', 'Being')
    commands.navigation.applyRenames([{ from: 'physics/Ontology.md', to: 'physics/Being.md' }])

    commands.palette.setTyped('Substance').catch(rethrow)

    expect(chooseRow(commands)?.target.path).toBe('physics/Being.md')
  })

  it('carries the tab holding it, so the invocation reaches it wherever it went', () => {
    const { commands, openNote } = createPalette()
    openNote('physics/Ontology.md', 'active')
    commands.palette.startCommand('app:note.rename', front())

    commands.palette.setTyped('Substance').catch(rethrow)

    expect(chooseRow(commands)?.note).toBe('active')
  })

  it('leaves a step over another note at its current position', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.rename', front())

    commands.navigation.applyRenames([{ from: 'Elsewhere.md', to: 'Moved.md' }])

    commands.palette.setTyped('Substance').catch(rethrow)

    expect(chooseRow(commands)?.target.path).toBe('physics/Ontology.md')
  })
})

describe('a command that was not offered over what it was requested over', () => {
  it('says the vault could not be opened', () => {
    const { commands } = createPalette({ isReady: false })

    expect(commands.palette.getObjection('app:file.remove', front({ isReady: false }))).toBe(
      words.noVault,
    )
  })

  it('says nothing in front is a note', () => {
    const { commands } = createPalette()

    expect(
      commands.palette.getObjection(
        'app:file.remove',
        front({ kind: 'document', path: '', title: '' }),
      ),
    ).toBe(words.noNote)
  })

  it('says nothing at all about one that was taken up', () => {
    const { commands } = createPalette()

    expect(commands.palette.getObjection('app:file.remove', front())).toBe('')
    expect(commands.palette.getObjection('nothing of the sort', front())).toBe('')
  })
})

describe('a command that asks for a note', () => {
  it('asks the vault for the names that match, and draws each note once', async () => {
    const { commands, requests } = createPalette({}, [
      name('physics/Entropy.md', 'Entropy'),
      name('physics/Entropy.md', 'Entropy', 'What it counts'),
      name('Order.md', 'Order'),
    ])
    commands.palette.startCommand('app:file.open#note', front())

    await commands.palette.setTyped('en')

    expect(requests).toStrictEqual(['en'])
    expect(commands.view.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics/Entropy.md',
      'Order.md',
    ])
  })

  it('carries the note that was picked, not the one that was in front', async () => {
    const { commands } = createPalette({}, [name('physics/Entropy.md', 'Entropy')])
    commands.palette.startCommand('app:file.open#note', front())
    await commands.palette.setTyped('en')

    expect(chooseRow(commands, 'physics/Entropy.md')).toStrictEqual({
      id: 'app:file.open',
      target: {
        path: 'physics/Entropy.md',
        vault: { id: 'physics', name: 'Physics' },
        title: 'Entropy',
        file: '',
        others: [],
        kind: 'note',
        tab: 'tab',
      },
      note: null,
      name: '',
    })
  })

  it('says the vault could not answer, in the window’s own words', async () => {
    vi.spyOn(console, 'error').mockImplementation(vi.fn())
    const activeTarget = ref(front())
    const commands = createCommandPalette({
      core: {
        names: async () => Promise.reject(new Error('no model is set')),
        vaults: async () => installation(),
      },
      words,
      at: () => activeTarget.value,
      knows: { getTitle: () => '', getTabAt: () => null },
      lists: { getStepGroups: () => [], previewItem: vi.fn() },
      runs: runSupport(),
      wait: vi.fn(() => Promise.resolve()),
      getCommands: createAppCommands,
    })
    commands.palette.setOpen(true)
    commands.palette.startCommand('app:file.open#note', front())

    await commands.palette.setTyped('en')

    expect(commands.view.groups.value[0]?.silence).toBe(words.noQuery)
  })
})

describe('a command that offers a list the window holds', () => {
  /** The themes, in the two groups they come off, and one group holding none. */
  const THEMES: readonly StepGroup[] = [
    {
      id: 'shipping',
      title: 'Built-in',
      items: [
        { id: 'preset:numen', title: 'numen', detail: 'Current' },
        { id: 'preset:dracula', title: 'dracula' },
      ],
    },
    { id: 'owned', title: 'Custom themes', items: [], silence: 'A .css file there is one' },
  ]

  /** The three halves, drawn and not to be chosen, as a pinned theme leaves them. */
  const HALVES: readonly StepGroup[] = [
    {
      id: 'half',
      title: 'Light and dark',
      items: [
        { id: 'mode:system', title: 'System', isDisabled: true },
        { id: 'mode:light', title: 'Light', detail: 'The applied theme sets this itself' },
      ],
    },
  ]

  const target = {
    [THEME_ID]: THEMES,
    [MODE_ID]: HALVES,
  }

  it('offers the groups the window holds, each under its own name', () => {
    const { commands } = createPalette({}, [], target)
    commands.palette.startCommand(THEME_ID, front())

    expect(commands.navigation.crumb.value).toBe(THEME_TEXT)
    expect(commands.navigation.placeholder.value).toBe(words.typeItems)
    expect(commands.view.groups.value.map((group) => group.title)).toStrictEqual([
      'Built-in',
      'Custom themes',
    ])
    expect(getItemIds(commands.view.groups)).toStrictEqual({
      shipping: ['preset:numen', 'preset:dracula'],
      owned: [],
    })
    expect(getEmptyMessage(commands.view.groups, 'owned')).toBe('A .css file there is one')
  })

  it('offers each command the list it holds for that command', () => {
    const { commands } = createPalette({}, [], target)
    commands.palette.startCommand(MODE_ID, front())

    expect(commands.navigation.crumb.value).toBe(MODE_TEXT)
    expect(getItemIds(commands.view.groups)).toStrictEqual({
      half: ['mode:system', 'mode:light'],
    })
  })

  it('keeps to the rows the words typed name, and marks where they match', async () => {
    const { commands } = createPalette({}, [], target)
    commands.palette.startCommand(THEME_ID, front())

    await commands.palette.setTyped('rac')

    expect(getItemIds(commands.view.groups)).toStrictEqual({
      shipping: ['preset:dracula'],
      owned: [],
    })
    expect(commands.view.groups.value[0]?.items[0]?.at).toStrictEqual([{ start: 1, end: 4 }])
  })

  it('draws a row the window says cannot be chosen, and chooses nothing by it', () => {
    const { commands } = createPalette({}, [], target)
    commands.palette.startCommand(MODE_ID, front())

    expect(commands.view.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(chooseRow(commands, 'mode:system')).toBeNull()
    expect(chooseRow(commands, 'mode:none')).toBeNull()
  })

  it('returns the row that was chosen, from whichever group it stood in', () => {
    const { commands } = createPalette({}, [], target)
    commands.palette.startCommand(THEME_ID, front())

    expect(chooseRow(commands, 'preset:dracula')).toStrictEqual({
      id: THEME_ID,
      target: {
        path: 'physics/Ontology.md',
        title: 'Ontology',
        file: '',
        others: [],
        kind: 'note',
        tab: 'tab',
        vault: { id: 'physics', name: 'Physics' },
      },
      note: null,
      name: 'preset:dracula',
    })
  })

  it('says which command the keyboard is standing in, and that it remains nowhere after', () => {
    const { commands, displayedLists } = createPalette({}, [], target)
    commands.palette.startCommand(MODE_ID, front())

    commands.palette.previewItem('mode:light')
    commands.navigation.leaveStep()
    commands.palette.startCommand(THEME_ID, front())
    commands.palette.previewItem('preset:dracula')
    commands.navigation.leaveStep()

    expect(displayedLists).toStrictEqual([
      `${MODE_ID} mode:light`,
      `${MODE_ID} `,
      `${THEME_ID} preset:dracula`,
      `${THEME_ID} `,
    ])
  })

  it('says nothing about where the keyboard is at a step that shows nothing', () => {
    const { commands, displayedLists } = createPalette({}, [], target)
    commands.palette.startCommand('app:note.createChild', front())

    commands.palette.previewItem('anything')
    commands.navigation.leaveStep()

    expect(displayedLists).toStrictEqual([])
  })
})

describe('leaving a step', () => {
  it('returns to the commands, and the palette stays open', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.createChild', front())

    commands.navigation.leaveStep()

    expect(commands.palette.open.value).toBe(true)
    expect(commands.view.groups.value.map((group) => group.id)).toStrictEqual([
      'note',
      'file',
      'window',
      'settings',
      'vault',
    ])
  })

  it('puts the commands away from the commands themselves', () => {
    const { commands } = createPalette()

    commands.navigation.leaveStep()

    expect(commands.palette.open.value).toBe(false)
  })

  it('passes the field back to the search from the commands themselves', () => {
    const { commands } = createPalette()

    expect(commands.navigation.goBack()).toBe(true)
    expect(commands.palette.open.value).toBe(false)
  })

  it('keeps the field on a step, and drops the step', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.createChild', front())

    expect(commands.navigation.goBack()).toBe(false)
    expect(commands.navigation.crumb.value).toBe(words.command)
  })

  it('is a step of its own to the palette, every time', () => {
    const { commands } = createPalette()
    const steps = [commands.navigation.step.value]
    commands.palette.startCommand('app:note.createChild', front())
    steps.push(commands.navigation.step.value)
    commands.navigation.leaveStep()
    steps.push(commands.navigation.step.value)
    commands.palette.startCommand('app:note.rename', front())
    steps.push(commands.navigation.step.value)

    expect(new Set(steps).size).toBe(3)
    expect(steps[1]).not.toBe(steps[3])
  })
})

describe('a command requested for from outside the palette', () => {
  it('carries out a command that needs nothing with no name', () => {
    const { commands } = createPalette()

    expect(commands.palette.startCommand('app:file.copyPath', front())?.name).toBe('')
  })

  it('starts from no step where the palette was shut with a step standing', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.rename', front())
    commands.palette.open.value = false

    commands.palette.startCommand('app:note.rename', front())
    commands.navigation.leaveStep()

    expect(commands.navigation.crumb.value).toBe(words.command)
  })

  it('opens the palette where it asks for what it needs', () => {
    const { commands } = createPalette()
    commands.palette.setOpen(false)

    expect(commands.palette.startCommand('app:note.rename', front())).toBeNull()
    expect(commands.palette.open.value).toBe(true)
    expect(commands.navigation.crumb.value).toBe(words.title)
  })

  it('leaves the palette shut where it needs nothing', () => {
    const { commands } = createPalette()
    commands.palette.setOpen(false)

    expect(commands.palette.startCommand('app:file.copyPath', front())?.id).toBe(
      'app:file.copyPath',
    )
    expect(commands.palette.open.value).toBe(false)
  })
})

describe('the commands over the vaults an installation holds', () => {
  /** Only renaming is over the vault in front. The rest ask for one first. */
  it('omits rename command when no vault is active', () => {
    const { commands } = createPalette({ vault: { id: '', name: '' } })

    expect(getItemIds(commands.view.groups).vault).toStrictEqual([
      'app:file.open#note',
      'app:vault.open',
      'app:vault.create',
      'app:vault.remove',
      'app:vault.erase',
      'app:trash.empty',
    ])
  })
})

describe('the trash emptied', () => {
  it('is confirmed over the vault in front, saying it cannot be undone', () => {
    const { commands } = createPalette()

    commands.palette.startCommand('app:trash.empty', front())

    expect(commands.view.groups.value[0]?.items.map((item) => item.title)).toStrictEqual([
      words.keepsTrash,
      'Empty the trash of “Physics”',
    ])
    expect(commands.view.groups.value[0]?.items[1]?.detail).toBe(words.exactly)
  })

  it('is carried out once the question is answered with yes', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:trash.empty', front())

    expect(chooseRow(commands, 1)).toMatchObject({ id: 'app:trash.empty' })
  })

  it('is left alone when answered with no', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:trash.empty', front())

    expect(chooseRow(commands, 0)).toBeNull()
  })
})

describe('a command that asks for a vault', () => {
  const two = () => installation(vault('physics', 'Physics'), vault('heat', 'Heat'))

  it('lists the vaults the installation holds, each at the folder it remains in', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.palette.startCommand('app:vault.open', front())

    await settle()

    expect(commands.view.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics',
      'heat',
    ])
    expect(commands.view.groups.value[0]?.items[1]?.detail).toBe('/vaults/Heat')
  })

  /** The folder may be on an unmounted volume, and the vault stays listed. */
  it('marks the vault whose folder is not there, and names where it looked', async () => {
    const { commands } = createPalette({}, [], {}, installation(vault('gone', 'Gone', true)))
    commands.palette.startCommand('app:vault.open', front())
    await settle()

    expect(commands.view.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.view.groups.value[0]?.items[0]?.detail).toBe(`${words.gone} · /vaults/Gone`)
    expect(chooseRow(commands, 'gone')).toBeNull()
  })

  /**
   * The window always relies on something, so the vault in front is not one
   * the list takes. It is drawn with the reason beside it, and choosing it
   * does nothing.
   */
  it('marks active vault and ignores duplicate selection', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.palette.startCommand('app:vault.open', front())
    await settle()

    expect(commands.view.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.view.groups.value[0]?.items[0]?.detail).toBe(
      `${words.current} · /vaults/Physics`,
    )
    expect(chooseRow(commands, 'physics')).toBeNull()
  })

  it('says on every row what choosing it asks for', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.palette.startCommand('app:vault.remove', front())
    await settle()

    expect(commands.view.groups.value[0]?.items[1]?.actions?.[0]?.text).toBe(words.removeVault)
  })

  it('uses selected vault over active vault', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.palette.startCommand('app:vault.open', front())
    await settle()

    expect(chooseRow(commands, 'heat')).toMatchObject({
      id: 'app:vault.open',
      target: { vault: { id: 'heat', name: 'Heat' } },
    })
  })

  it('keeps the vaults the words typed name, so the field means something', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.palette.startCommand('app:vault.open', front())
    await settle()

    await commands.palette.setTyped('hea')

    expect(commands.view.groups.value[0]?.items.map((item) => item.id)).toStrictEqual(['heat'])
  })

  it('says the list could not be requested, in the window’s own words', async () => {
    vi.spyOn(console, 'error').mockImplementation(vi.fn())
    const activeTarget = ref(front())
    const commands = createCommandPalette({
      core: {
        names: async () => [],
        vaults: async () => Promise.reject(new Error('the list is not there')),
      },
      words,
      at: () => activeTarget.value,
      knows: { getTitle: () => '', getTabAt: () => null },
      lists: { getStepGroups: () => [], previewItem: vi.fn() },
      runs: runSupport(),
      wait: vi.fn(() => Promise.resolve()),
      getCommands: createAppCommands,
    })
    commands.palette.setOpen(true)

    commands.palette.startCommand('app:vault.open', front())
    await settle()

    expect(commands.view.groups.value[0]?.silence).toBe(words.noQuery)
  })
})

describe('renaming the vault in front', () => {
  it('remains filled with the name the vault has now', () => {
    const { commands } = createPalette()

    commands.palette.startCommand('app:vault.rename', front())

    expect(commands.navigation.crumb.value).toBe(words.renameVault)
    expect(commands.palette.typed.value).toBe('Physics')
  })

  it('passes name to active vault', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:vault.rename', front())
    commands.palette.setTyped('Heat').catch(rethrow)

    expect(chooseRow(commands)).toMatchObject({
      id: 'app:vault.rename',
      name: 'Heat',
      target: { vault: { id: 'physics', name: 'Physics' } },
    })
  })
})

/**
 * The window always relies on a vault, so neither of these is over the one in
 * front. Both ask for a vault first, and everything they say after is about
 * the vault that was chosen.
 */
describe('a vault taken off the list', () => {
  /** The list opened, and the vault the window is not showing chosen off it. */
  const chooseVault = async (id: string) => {
    const { commands } = createPalette(
      {},
      [],
      {},
      installation(vault('physics', 'Physics'), vault('heat', 'Heat')),
    )
    commands.palette.startCommand(id, front())
    await settle()
    chooseRow(commands, 'heat')
    return commands
  }

  it('is confirmed over the vault that was chosen, saying the folder stays', async () => {
    const commands = await chooseVault('app:vault.remove')

    expect(commands.view.groups.value[0]?.items.map((item) => item.title)).toStrictEqual([
      words.keepsVault,
      'Remove “Heat”',
    ])
    expect(commands.view.groups.value[0]?.items[0]?.title).toBe(words.keepsVault)
    expect(commands.view.groups.value[0]?.items[1]?.title).toBe('Remove “Heat”')
    expect(commands.view.groups.value[0]?.items[1]?.detail).toBe(words.stays)
  })

  it('is carried out over that vault once the question is answered', async () => {
    const commands = await chooseVault('app:vault.remove')

    expect(chooseRow(commands, 1)).toMatchObject({
      id: 'app:vault.remove',
      target: { vault: { id: 'heat', name: 'Heat' } },
    })
  })

  it('pops the confirmation step and returns null when answered with no', async () => {
    const commands = await chooseVault('app:vault.remove')

    expect(chooseRow(commands, 0)).toBeNull()
    expect(commands.view.groups.value[0]?.id).toBe('vaults')
  })

  it('returns the list, with the vaults on it, where the step is left', async () => {
    const commands = await chooseVault('app:vault.remove')

    commands.navigation.leaveStep()
    await settle()

    expect(commands.view.groups.value[0]?.id).toBe('vaults')
    expect(commands.view.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics',
      'heat',
    ])
  })

  it('waits for the name of the vault chosen, and says where its folder is located', async () => {
    const commands = await chooseVault('app:vault.erase')

    expect(commands.view.groups.value[0]?.items[0]?.title).toBe('Erase “Heat”')
    expect(commands.view.groups.value[0]?.items[0]?.detail).toBe(words.binned)
    expect(commands.view.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.navigation.placeholder.value).toBe(words.typeVaultBack)
    expect(chooseRow(commands)).toBeNull()
  })

  it('submits once the name of that vault is what was typed', async () => {
    const commands = await chooseVault('app:vault.erase')

    commands.palette.setTyped('Heat').catch(rethrow)

    expect(commands.view.groups.value[0]?.items[0]?.isDisabled).toBe(false)
    expect(chooseRow(commands)).toMatchObject({
      id: 'app:vault.erase',
      target: { vault: { id: 'heat', name: 'Heat' } },
    })
  })

  it('ignores active vault name', async () => {
    const commands = await chooseVault('app:vault.erase')

    commands.palette.setTyped('Physics').catch(rethrow)

    expect(chooseRow(commands)).toBeNull()
  })
})

const createSlowPalette = (commandLists: Record<string, readonly StepGroup[]> = {}) => {
  const names: Deferred<readonly NameMatch[]>[] = []
  const lists: Deferred<VaultList>[] = []
  const holds: Deferred<void>[] = []
  const request: string[] = []
  const signals: (AbortSignal | undefined)[] = []
  const core = {
    names: (query: string, _limit: number, signal?: AbortSignal) => {
      request.push(query)
      signals.push(signal)
      const one = createDeferred<readonly NameMatch[]>()
      names.push(one)
      return one.promise
    },
    vaults: () => {
      const one = createDeferred<VaultList>()
      lists.push(one)
      return one.promise
    },
  }
  const hold = { isOn: false }
  const wait = () => {
    if (!hold.isOn) return Promise.resolve()
    const one = createDeferred<void>()
    holds.push(one)
    return one.promise
  }
  const commands = createCommandPalette({
    core,
    words,
    at: () => front(),
    knows: createLookup().knows,
    lists: createLists(commandLists).paletteLists,
    runs: runSupport(),
    wait,
    getCommands: createAppCommands,
  })
  return { commands, names, lists, holds, hold, requests: request, signals }
}

const describeGroup = (groups: ReturnType<typeof createPalette>['commands']['view']['groups']) => {
  const group = groups.value[0]
  return {
    items: group?.items.map((item) => item.id),
    isWorking: group?.isWorking,
    silence: group?.silence,
  }
}

describe('the palette before it is first opened', () => {
  it('holds nothing typed', () => {
    const { commands } = createSlowPalette()

    expect(commands.palette.typed.value).toBe('')
  })
})

describe('a step that asks for a note, as the vault answers', () => {
  it('opens working at nothing and asking for a note', () => {
    const { commands } = createSlowPalette()

    commands.palette.startCommand('app:file.open#note', front())

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.typeNote,
    })
  })

  it('opens saying nothing of a list a step before it could not ask', async () => {
    const { commands, lists } = createSlowPalette()
    commands.palette.startCommand('app:vault.open', front())
    lists[0]?.fail('the list is not there')
    await settle()
    commands.navigation.leaveStep()

    commands.palette.startCommand('app:file.open#note', front())

    expect(getEmptyMessage(commands.view.groups, 'picking')).toBe(words.typeNote)
  })

  it('is working while the vault has not answered', async () => {
    const { commands } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())

    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: [],
      isWorking: true,
      silence: words.noneFound,
    })
  })

  it('stops working once the vault answers', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: ['physics/Entropy.md'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('asks about the words and not the spaces around them', async () => {
    const { commands, requests } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())

    commands.palette.setTyped('  en  ').catch(rethrow)
    await settle()

    expect(requests).toStrictEqual(['en'])
  })

  it('asks nothing once what was typed is taken back, and draws nothing found', async () => {
    const { commands, names, requests } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('en').catch(rethrow)
    await settle()
    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    commands.palette.setTyped('').catch(rethrow)
    await settle()

    expect(requests).toStrictEqual(['en'])
    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.typeNote,
    })
  })

  it('asks nothing about a keystroke another one overtook while it waited', async () => {
    const { commands, holds, hold, requests } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    hold.isOn = true

    commands.palette.setTyped('e').catch(rethrow)
    commands.palette.setTyped('en').catch(rethrow)
    holds[0]?.answer()
    await settle()

    expect(requests).toStrictEqual([])

    holds[1]?.answer()
    await settle()

    expect(requests).toStrictEqual(['en'])
  })

  it('stops the name question a newer query overtakes', async () => {
    const { commands, signals } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('e').catch(rethrow)
    await settle()

    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    expect(signals.map((one) => one?.aborted)).toStrictEqual([true, false])
  })

  it('stops the name question when the palette closes', async () => {
    const { commands, signals } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    commands.palette.setOpen(false)

    expect(signals.map((one) => one?.aborted)).toStrictEqual([true])
  })

  it('draws the answer to the question requested last, whatever lands after it', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('e').catch(rethrow)
    await settle()
    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    names[1]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()
    names[0]?.answer([name('Ether.md', 'Ether')])
    await settle()

    expect(describeGroup(commands.view.groups).items).toStrictEqual(['physics/Entropy.md'])
  })

  it('draws no failure of a superseded query', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('e').catch(rethrow)
    await settle()
    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    names[1]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()
    names[0]?.fail('the vault went away')
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: ['physics/Entropy.md'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('keeps working where only a superseded query has a result', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('e').catch(rethrow)
    await settle()
    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    names[0]?.answer([name('Ether.md', 'Ether')])
    await settle()

    expect(describeGroup(commands.view.groups).isWorking).toBe(true)
  })

  it('holds nothing where the vault could not answer', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    names[0]?.fail('no model is set')
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.noQuery,
    })
  })

  it('says nothing of a failure while the next question is out', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('e').catch(rethrow)
    await settle()
    names[0]?.fail('no model is set')
    await settle()

    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    expect(getEmptyMessage(commands.view.groups, 'picking')).toBe(words.noneFound)
  })

  it('draws no answer to a question requested at a step that was left', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('en').catch(rethrow)
    await settle()
    commands.navigation.leaveStep()
    commands.palette.startCommand('app:file.open#note', front())

    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    expect(describeGroup(commands.view.groups).items).toStrictEqual([])
  })

  it('knows no note of an answer that lands after the palette is put away', async () => {
    const { commands, names } = createSlowPalette()
    commands.palette.setOpen(true)
    commands.palette.startCommand('app:file.open#note', front())
    commands.palette.setTyped('en').catch(rethrow)
    await settle()

    commands.palette.setOpen(false)
    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    expect(commands.view.findTabKind('physics/Entropy.md')).toBeNull()
  })

  it('asks the vault nothing at a step that does not pick a note', async () => {
    const { commands, requests } = createSlowPalette()
    commands.palette.startCommand('app:note.createChild', front())

    commands.palette.setTyped('Entropy').catch(rethrow)
    await settle()

    expect(requests).toStrictEqual([])
  })
})

describe('a step that asks for a vault, as the list answers', () => {
  const two = () => installation(vault('physics', 'Physics'), vault('heat', 'Heat'))

  it('holds nothing and is working while the list has not returned', async () => {
    const { commands } = createSlowPalette()

    commands.palette.startCommand('app:vault.open', front())
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: [],
      isWorking: true,
      silence: words.noneFound,
    })
  })

  it('draws the list requested for last, and is working until that one is back', async () => {
    const { commands, lists } = createSlowPalette()
    commands.palette.startCommand('app:vault.open', front())
    commands.navigation.leaveStep()
    commands.palette.startCommand('app:vault.open', front())

    lists[0]?.answer(two())
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: [],
      isWorking: true,
      silence: words.noneFound,
    })

    lists[1]?.answer(installation(vault('heat', 'Heat')))
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: ['heat'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('draws no failure of a superseded list request', async () => {
    const { commands, lists } = createSlowPalette()
    commands.palette.startCommand('app:vault.open', front())
    commands.navigation.leaveStep()
    commands.palette.startCommand('app:vault.open', front())

    lists[1]?.answer(two())
    await settle()
    lists[0]?.fail('the list is not there')
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: ['physics', 'heat'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('holds nothing where the list could not be requested', async () => {
    const { commands, lists } = createSlowPalette()
    commands.palette.startCommand('app:vault.open', front())

    lists[0]?.fail('the list is not there')
    await settle()

    expect(describeGroup(commands.view.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.noQuery,
    })
  })

  it('keeps the vaults the words typed name, around the spaces typed', async () => {
    const { commands, lists } = createSlowPalette()
    commands.palette.startCommand('app:vault.open', front())
    lists[0]?.answer(two())
    await settle()

    await commands.palette.setTyped('  hea  ')

    expect(describeGroup(commands.view.groups).items).toStrictEqual(['heat'])
  })
})

describe('a command requested for while a step remains open', () => {
  it('remains over that step, and leaving it returns to it', () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:note.createChild', front())

    commands.palette.startCommand('app:note.rename', front())
    commands.navigation.leaveStep()

    expect(commands.navigation.crumb.value).toBe(words.child)
  })
})

describe('what a step draws', () => {
  it('lists each found note by title, highlights name matches and falls back to the file name', async () => {
    const { commands } = createPalette({}, [
      name('physics/Entropy.md', 'Entropy'),
      name('physics/Heat.md', 'Heat', 'What it counts'),
      name('Untitled.md', ''),
    ])
    commands.palette.startCommand('app:file.open#note', front())

    await commands.palette.setTyped('en')

    expect(commands.view.groups.value[0]?.id).toBe('picking')
    expect(commands.view.groups.value[0]?.items).toStrictEqual([
      {
        id: 'physics/Entropy.md',
        title: 'Entropy',
        at: [{ start: 0, end: 1 }],
        actions: [{ id: expect.any(String), text: words.travel }],
      },
      {
        id: 'physics/Heat.md',
        title: 'Heat',
        actions: [{ id: expect.any(String), text: words.travel }],
      },
      {
        id: 'Untitled.md',
        title: 'Untitled.md',
        at: [{ start: 0, end: 1 }],
        actions: [{ id: expect.any(String), text: words.travel }],
      },
    ])
  })

  it('says which tab kind opens each note found, and nothing of a note it did not find', async () => {
    const { commands } = createPalette({}, [
      name('physics/Entropy.md', 'Entropy', '', 'sample'),
      name('physics/Heat.md', 'Heat', '', 'form'),
    ])
    commands.palette.startCommand('app:file.open#note', front())

    await commands.palette.setTyped('en')

    expect(commands.view.findTabKind('physics/Heat.md')).toBe('form')
    expect(commands.view.findTabKind('physics/Entropy.md')).toBe('sample')
    expect(commands.view.findTabKind('Order.md')).toBeNull()
  })

  it('asks for a note where only spaces are typed', async () => {
    const { commands } = createPalette()
    commands.palette.startCommand('app:file.open#note', front())

    await commands.palette.setTyped('   ')

    expect(getEmptyMessage(commands.view.groups, 'picking')).toBe(words.typeNote)
  })

  it('confirms under the answer each row gives, filtered by the words typed around spaces', async () => {
    const { commands } = createPalette(
      {},
      [],
      {},
      installation(vault('physics', 'Physics'), vault('heat', 'Heat')),
    )
    commands.palette.startCommand('app:vault.remove', front())
    await settle()
    chooseRow(commands, 'heat')

    expect(
      commands.view.groups.value[0]?.items.map((item) =>
        (item.actions ?? []).map((action) => action.text),
      ),
    ).toStrictEqual([[words.keepsVault], [words.removeVaultAction]])

    commands.palette.setTyped('  remo  ').catch(rethrow)

    expect(commands.view.groups.value[0]?.items.map((item) => item.title)).toStrictEqual([
      'Remove “Heat”',
    ])
  })
})

describe('a row of a list the window holds', () => {
  const THEMES: readonly StepGroup[] = [
    {
      id: 'shipping',
      title: 'Built-in',
      items: [
        { id: 'preset:numen', title: 'numen', detail: 'Current' },
        { id: 'preset:dracula', title: 'dracula' },
      ],
    },
  ]

  it('is lit nowhere while nothing is typed', () => {
    const { commands } = createPalette({}, [], { [THEME_ID]: THEMES })
    commands.palette.startCommand(THEME_ID, front())

    expect(commands.view.groups.value[0]?.items).toStrictEqual([
      {
        id: 'preset:numen',
        title: 'numen',
        detail: 'Current',
        actions: [{ id: expect.any(String), text: words.chooses }],
      },
      {
        id: 'preset:dracula',
        title: 'dracula',
        actions: [{ id: expect.any(String), text: words.chooses }],
      },
    ])
  })

  it('is lit from its first letter where the words typed begin it', async () => {
    const { commands } = createPalette({}, [], { [THEME_ID]: THEMES })
    commands.palette.startCommand(THEME_ID, front())

    await commands.palette.setTyped('num')

    expect(commands.view.groups.value[0]?.items.map((item) => item.at)).toStrictEqual([
      [{ start: 0, end: 3 }],
    ])
  })

  it('is kept by the words typed around spaces', async () => {
    const { commands } = createPalette({}, [], { [THEME_ID]: THEMES })
    commands.palette.startCommand(THEME_ID, front())

    await commands.palette.setTyped('  rac  ')

    expect(commands.view.groups.value[0]?.items.map((item) => item.at)).toStrictEqual([
      [{ start: 1, end: 4 }],
    ])
  })
})

describe('the icon of a command', () => {
  it('is the one its declaration gives, and none for a command that gives none', () => {
    const icon = defineComponent({ name: 'GraphIcon', render: () => null })
    const commands = createCommandPalette({
      core: {
        names: async () => [],
        vaults: async () => installation(vault('physics', 'Physics')),
      },
      words,
      at: () => front(),
      knows: createLookup().knows,
      lists: createLists({}).paletteLists,
      runs: runSupport(),
      wait: vi.fn(() => Promise.resolve()),
      getCommands: () => [
        ...createAppCommands(),
        {
          id: 'org.example.sample:graph.run',
          text: 'New graph',
          icon,
          group: 'window',
          isAvailable: () => true,
        },
      ],
    })

    expect(commands.view.findCommandIcon('org.example.sample:graph.run')).toBe(icon)
    expect(commands.view.findCommandIcon('app:file.open#document')).toBeNull()
  })
})

describe('runSupport, arguments steps, and palette shortcuts', () => {
  it('starts a hidden command given values', () => {
    const commands = createCommandPalette({
      core: {
        names: async () => [],
        vaults: async () => installation(vault('physics', 'Physics')),
      },
      words,
      at: () => front(),
      knows: createLookup().knows,
      lists: createLists({}).paletteLists,
      runs: runSupport(),
      getCommands: () => [
        { id: 'app:test.reveal', text: 'Reveal', group: 'note', isAvailable: () => false },
      ],
    })

    expect(commands.palette.startCommand('app:test.reveal', front())).toBeNull()
    expect(
      commands.palette.startCommand('app:test.reveal', front(), undefined, { kind: 'plex' }),
    ).toMatchObject({ id: 'app:test.reveal' })
  })

  it('tracks unsupported runs in runSupport', () => {
    const runs = runSupport()
    expect(runs.canRun('ocr')).toBe(true)
    runs.cannotRun('ocr')
    expect(runs.canRun('ocr')).toBe(false)
    expect(runs.canRun('other')).toBe(true)
  })

  it('walks through multi-prompt arguments steps and handles commands without prompts', () => {
    const commands = createCommandPalette({
      core: {
        names: async () => [],
        vaults: async () => installation(vault('physics', 'Physics')),
      },
      words,
      at: () => front(),
      knows: createLookup().knows,
      lists: createLists({}).paletteLists,
      runs: runSupport(),
      getCommands: () => [
        {
          id: 'org.example.sample:args.run',
          text: 'With args',
          group: 'window',
          input: 'arguments',
          prompts: [
            { name: 'topic', title: 'Topic title', isRequired: true },
            { name: 'detail', title: 'Detail title', isRequired: false },
          ],
          isAvailable: () => true,
        },
        {
          id: 'org.example.sample:noPrompts.run',
          text: 'No prompts',
          group: 'window',
          input: 'arguments',
          isAvailable: () => true,
        },
      ],
    })

    commands.palette.startCommand('org.example.sample:noPrompts.run', front())
    expect(commands.navigation.placeholder.value).toBe(words.typeCommand)
    expect(chooseRow(commands)).toBeNull()
    commands.navigation.leaveStep()

    commands.palette.startCommand('org.example.sample:args.run', front())
    expect(commands.navigation.placeholder.value).toBe('Topic title')
    expect(chooseRow(commands)).toBeNull()

    commands.palette.setTyped('  Quantum  ').catch(rethrow)
    expect(chooseRow(commands)).toBeNull()
    expect(commands.navigation.placeholder.value).toBe('Detail title')

    const result = chooseRow(commands)
    expect(result?.id).toBe('org.example.sample:args.run')
    expect(result?.values).toStrictEqual({ topic: 'Quantum' })
  })

  it('asks only for the arguments the caller did not supply and keeps the supplied ones', () => {
    const commands = createCommandPalette({
      core: {
        names: async () => [],
        vaults: async () => installation(vault('physics', 'Physics')),
      },
      words,
      at: () => front(),
      knows: createLookup().knows,
      lists: createLists({}).paletteLists,
      runs: runSupport(),
      getCommands: () => [
        {
          id: 'org.example.sample:args.run',
          text: 'With args',
          group: 'window',
          input: 'arguments',
          prompts: [
            { name: 'topic', title: 'Topic title', isRequired: true },
            { name: 'detail', title: 'Detail title', isRequired: true },
          ],
          isAvailable: () => true,
        },
      ],
    })

    const all = commands.palette.startCommand('org.example.sample:args.run', front(), undefined, {
      topic: 'Quantum',
      detail: 'Spin',
    })
    expect(all?.id).toBe('org.example.sample:args.run')

    commands.palette.startCommand('org.example.sample:args.run', front(), undefined, {
      detail: 'Spin',
    })
    expect(commands.navigation.placeholder.value).toBe('Topic title')
    commands.palette.setTyped('Quantum').catch(rethrow)
    const result = chooseRow(commands)
    expect(result?.values).toStrictEqual({ detail: 'Spin', topic: 'Quantum' })
  })
})

describe('a command started with the text typed for it', () => {
  const NAMING = [
    'app:note.create',
    'app:note.createChild',
    'app:note.createParent',
    'app:note.createJump',
  ] as const

  it.each(NAMING)('runs %s at once under the trimmed text and opens no step', (id) => {
    const { commands } = createPalette()
    const stepBefore = commands.navigation.step.value
    const crumbBefore = commands.navigation.crumb.value

    const invocation = commands.palette.startCommand(id, front(), '  Entropy  ')

    expect(invocation).toMatchObject({ id, name: 'Entropy', target: { kind: 'note', tab: 'tab' } })
    expect(commands.navigation.step.value).toBe(stepBefore)
    expect(commands.navigation.crumb.value).toBe(crumbBefore)
  })

  it('attaches a relation to the note it is started over', () => {
    const { commands } = createPalette()

    for (const id of NAMING.slice(1)) {
      expect(commands.palette.startCommand(id, front(), 'Entropy')?.target.path).toBe(
        'physics/Ontology.md',
      )
    }
  })

  it('follows the tab that holds the note it is started over', () => {
    const { commands, openNote } = createPalette()
    openNote('physics/Ontology.md', 'active')

    expect(commands.palette.startCommand('app:note.createChild', front(), 'Entropy')?.note).toBe(
      'active',
    )
  })

  it('asks for the name as it did where the text typed is empty', () => {
    const { commands } = createPalette()

    expect(commands.palette.startCommand('app:note.createChild', front(), '')).toBeNull()
    expect(commands.navigation.crumb.value).toBe(words.child)
  })

  it('asks for the name as it did where the text typed is only blank', () => {
    const { commands } = createPalette()

    expect(commands.palette.startCommand('app:note.createChild', front(), '   ')).toBeNull()
    expect(commands.navigation.crumb.value).toBe(words.child)
  })

  it('runs nothing and opens no step where the command is unavailable', () => {
    const { commands } = createPalette()
    const crumbBefore = commands.navigation.crumb.value

    expect(
      commands.palette.startCommand(
        'app:note.createChild',
        front({ path: '', title: '' }),
        'Entropy',
      ),
    ).toBeNull()
    expect(commands.navigation.crumb.value).toBe(crumbBefore)
  })

  it('runs nothing over a window with no vault ready, and says so', () => {
    const { commands } = createPalette()

    expect(
      commands.palette.startCommand('app:note.create', front({ isReady: false }), 'Entropy'),
    ).toBeNull()
    expect(commands.palette.getObjection('app:note.create', front({ isReady: false }))).toBe(
      words.noVault,
    )
  })

  it('leaves a command that needs no name as it was, with no name', () => {
    const { commands } = createPalette()

    expect(commands.palette.startCommand('app:file.copyPath', front(), 'Entropy')?.name).toBe('')
  })

  it('still opens the step of a command that asks for a choice', () => {
    const { commands } = createPalette({}, [], {
      [THEME_ID]: [
        { id: 'themes', title: 'Themes', items: [{ id: 'preset:numen', title: 'numen' }] },
      ],
    })

    expect(commands.palette.startCommand(THEME_ID, front(), 'dark')).toBeNull()
    expect(commands.navigation.step.value).not.toBe('')
  })
})

describe('the command that opens a document chosen as a result', () => {
  const createDocument = (override: Partial<CommandTarget> = {}): CommandTarget =>
    front({ tab: '', kind: null, path: '', title: 'Atlas', file: 'library/a.bin', ...override })

  it('starts over a chosen document and carries its file, its title and its selector', () => {
    const { commands } = createPalette()

    expect(
      commands.palette.startCommand(
        'app:file.open#document',
        createDocument({ selector: { page: 3 } }),
      ),
    ).toMatchObject({
      id: 'app:file.open',
      target: {
        path: '',
        file: 'library/a.bin',
        title: 'Atlas',
        selector: { page: 3 },
      },
    })
  })

  it('starts over nothing that is in front, a note, or a window without a vault', () => {
    const { commands } = createPalette()

    expect(
      commands.palette.startCommand(
        'app:file.open#document',
        createDocument({ tab: 'tab', kind: 'reader' }),
      ),
    ).toBeNull()
    expect(
      commands.palette.startCommand(
        'app:file.open#document',
        createDocument({ path: 'a.md', file: 'a.md' }),
      ),
    ).toBeNull()
    expect(
      commands.palette.startCommand('app:file.open#document', createDocument({ isReady: false })),
    ).toBeNull()
    expect(
      commands.palette.startCommand('app:file.open#document', createDocument({ file: '' })),
    ).toBeNull()
  })

  it('says why it did not start, as any command does', () => {
    const { commands } = createPalette()

    expect(
      commands.palette.getObjection(
        'app:file.open#document',
        createDocument({ tab: 'tab', kind: 'reader' }),
      ),
    ).toBe(words.noNote)
    expect(
      commands.palette.getObjection('app:file.open#document', createDocument({ isReady: false })),
    ).toBe(words.noVault)
    expect(commands.palette.getObjection('app:file.open#document', createDocument())).toBe('')
  })

  it('is not in the list of commands over a window holding a document in a tab', () => {
    const { commands } = createPalette({ kind: 'reader', path: '', title: '', file: 'a.pdf' })

    expect(Object.values(getItemIds(commands.view.groups)).flat()).not.toContain(
      'app:file.open#document',
    )
  })
})
