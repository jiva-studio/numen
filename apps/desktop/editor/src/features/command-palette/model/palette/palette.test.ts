/**
 * Palette step requests and the active step's rendering, without a window.
 *
 * Covered: a tab holding a non-note is offered no note actions, and a
 * destructive action is reached only by typing the note's name.
 */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { runSupport } from './runSupport'
import type { CommandTarget, NoteLookup, PaletteLists, RunSupport, StepGroup } from '../../types'
import { useCommandPalette } from './palette'
import { usePaletteShortcuts } from './shortcuts'
import type { VaultList, Vault } from '@/entities/vault'
import type { NameMatch, SearchState } from '../search/search'
import { WORDS as words } from '@/shared/words'
import { createDeferred, type Deferred } from '@/testing/deferred'

/** Throws the rejection reason of a detached promise, so the test run reports it as an unhandled rejection. */
function rethrow(error: unknown): never {
  throw error
}

/** What is in front, which a test moves under the commands. */
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

/** One name as the vault answers one. */
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
})

/** One vault as the list answers one. */
const vault = (id: string, name: string, isMissing = false): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: isMissing,
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

/** The step lists of the window and every row previewed by a step. */
const createLists = (offers: Record<string, readonly StepGroup[]>) => {
  const lists = ref(offers)
  const displayedItems: string[] = []
  const paletteLists: PaletteLists = {
    getStepGroups: (command) => lists.value[command] ?? [],
    previewItem: (command, item) => {
      displayedItems.push(`${command} ${item}`)
    },
  }
  return { paletteLists, lists, displayedLists: displayedItems }
}

/** The commands over the test's current state, without a pending action. */
const createPalette = (
  override: Partial<CommandTarget> = {},
  found: readonly NameMatch[] = [],
  offers: Record<string, readonly StepGroup[]> = {},
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
  const retainedItem = createLists(offers)
  const commands = useCommandPalette(
    core,
    words,
    () => activeTarget.value,
    window.knows,
    retainedItem.paletteLists,
    runs,
    vi.fn(() => Promise.resolve()),
  )
  commands.setOpen(true)
  return {
    commands,
    at: activeTarget,
    target: activeTarget,
    requests: request,
    ...window,
    ...retainedItem,
  }
}

/** A moment for the list of vaults to come back. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const getItemIds = (groups: ReturnType<typeof createPalette>['commands']['groups']) =>
  Object.fromEntries(groups.value.map((group) => [group.id, group.items.map((item) => item.id)]))

/** The message one group shows while it has no items. */
const getEmptyMessage = (
  groups: ReturnType<typeof createPalette>['commands']['groups'],
  id: string,
) => groups.value.find((group) => group.id === id)?.silence

describe('the commands as they open', () => {
  it('offers everything that can be done to what is in front, with nothing typed', () => {
    const { commands } = createPalette()

    expect(getItemIds(commands.groups)).toStrictEqual({
      note: [
        'app:note.open',
        'app:note.create-child',
        'app:note.create-parent',
        'app:note.create-jump',
        'app:note.rename',
        'app:note.trash',
        'app:note.copy-path',
      ],
      window: ['app:note.create', 'app:tab.close', 'app:vault.search'],
      settings: [
        'app:appearance.choose-theme',
        'app:appearance.choose-mode',
        'app:appearance.choose-interface-size',
        'app:appearance.choose-text-size',
        'app:note.choose-title-sync',
        'app:performance.choose-profile',
        'app:settings.open',
      ],
      vault: [
        'app:note.go-to-start',
        'app:note.go-to',
        'app:vault.open',
        'app:vault.create',
        'app:vault.rename',
        'app:vault.remove',
        'app:vault.erase',
      ],
    })
  })

  it('draws the command reached by Shift and Enter on the row that names it', () => {
    const { commands } = createPalette()
    const items = commands.groups.value[0]?.items ?? []

    expect(
      items.find((item) => item.id === 'app:note.open')?.actions?.map((one) => one.id),
    ).toStrictEqual(['app:note.open', 'app:note.open-beside'])
    expect(
      items.find((item) => item.id === 'app:note.trash')?.actions?.map((one) => one.id),
    ).toStrictEqual(['app:note.trash', 'app:note.delete'])
  })

  it('says which key reaches a command away from the palette', () => {
    const { commands } = createPalette()
    const window = commands.groups.value[1]?.items ?? []

    expect(window.find((item) => item.id === 'app:vault.search')?.keys).toBe(words.findKeys)
  })

  it('writes nothing under a command that its whole group is over', () => {
    const { commands } = createPalette()

    for (const item of commands.groups.value[0]?.items ?? []) expect(item.detail).toBeUndefined()
  })

  it('keeps the commands the words typed leave, and lights where they stand', () => {
    const { commands } = createPalette()

    commands.setTyped('child').catch(rethrow)

    expect(getItemIds(commands.groups)).toStrictEqual({ note: ['app:note.create-child'] })
    expect(commands.groups.value[0]?.items[0]?.at).toStrictEqual([{ start: 7, end: 12 }])
  })
})

describe('what is in front', () => {
  it('offers nothing over a note when a document is in front', () => {
    const { commands } = createPalette({ kind: 'document', path: '', title: '' })

    expect(getItemIds(commands.groups).note).toBeUndefined()
  })

  it('offers nothing over a note when the tab in front is of no kind', () => {
    const { commands } = createPalette({ kind: null, path: '', title: '' })

    expect(getItemIds(commands.groups).note).toBeUndefined()
  })

  it('offers nothing over a note when the plex is standing nowhere', () => {
    const { commands } = createPalette({ kind: 'plex', path: '', title: '' })

    expect(getItemIds(commands.groups).note).toBeUndefined()
  })

  /** A vault that cannot be opened must still be possible to leave. */
  it('says the vault could not be opened, and offers nothing over the note', () => {
    const { commands } = createPalette({ isReady: false })

    expect(getItemIds(commands.groups)).toStrictEqual({
      window: ['app:tab.close', 'app:vault.search'],
      settings: [
        'app:appearance.choose-theme',
        'app:appearance.choose-mode',
        'app:appearance.choose-interface-size',
        'app:appearance.choose-text-size',
        'app:note.choose-title-sync',
        'app:performance.choose-profile',
        'app:settings.open',
      ],
      vault: [
        'app:vault.open',
        'app:vault.create',
        'app:vault.rename',
        'app:vault.remove',
        'app:vault.erase',
      ],
    })
    expect(commands.groups.value.some((g) => g.id === 'note')).toBe(false)
  })

  it('offers no close where the window holds no tab at all', () => {
    const { commands } = createPalette({ tab: '' })

    expect(getItemIds(commands.groups).window).not.toContain('app:tab.close')
  })

  it('follows what the person is looking at while it is open', () => {
    const { commands, target: activeTarget } = createPalette()

    activeTarget.value = front({ kind: 'document', path: '', title: '' })

    expect(getItemIds(commands.groups).note).toBeUndefined()
  })
})

describe('a command that needs nothing', () => {
  it('is carried out the moment it is chosen, over the note in front', () => {
    const { commands } = createPalette()

    expect(commands.chooseItem('app:note.open', 'app:note.open')).toStrictEqual({
      id: 'app:note.open',
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
    const { commands, openNote } = createPalette()
    openNote('physics/Ontology.md', 'active')

    expect(commands.chooseItem('app:note.open', 'app:note.open')?.note).toBe('active')
  })

  it('is the second one where Shift and Enter reached it', () => {
    const { commands } = createPalette()

    expect(commands.chooseItem('app:note.open', 'app:note.open-beside')?.id).toBe(
      'app:note.open-beside',
    )
  })

  it('is nothing where what is in front does not offer it', () => {
    const { commands } = createPalette({ kind: 'document', path: '', title: '' })

    expect(commands.chooseItem('app:note.open', 'app:note.open')).toBeNull()
  })
})

describe('a command that asks for a name', () => {
  it('relies on a step of its own, and says which step it is', () => {
    const { commands } = createPalette()

    expect(commands.startCommand('app:note.create-child', front())).toBeNull()
    expect(commands.crumb.value).toBe(words.child)
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['naming'])
  })

  it('offers what was typed as the name, and nothing before anything is', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.create-child', front())

    expect(commands.groups.value[0]?.items).toStrictEqual([])

    commands.setTyped('  Entropy  ').catch(rethrow)

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Name it “Entropy”')
  })

  it('carries the name to the note it was requested over', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.create-child', front())
    commands.setTyped('Entropy').catch(rethrow)

    expect(commands.chooseItem('name', 'name')).toStrictEqual({
      id: 'app:note.create-child',
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
    const { commands } = createPalette()
    commands.startCommand('app:note.create-child', front())

    expect(commands.chooseItem('name', 'name')).toBeNull()
  })

  it('puts the title the note has now in the field, for a change of title', () => {
    const { commands } = createPalette()

    commands.startCommand('app:note.rename', front())

    expect(commands.typed.value).toBe('Ontology')
  })
})

/** The note moves to the vault's .trash folder. */
describe('removing a note', () => {
  it('is carried out the moment it is requested for, over the note in front', () => {
    const { commands } = createPalette()

    const invocation = commands.startCommand('app:note.trash', front())

    expect(invocation?.id).toBe('app:note.trash')
    expect(invocation?.path).toBe('physics/Ontology.md')
  })

  it('carries every file it was over into the invocation', () => {
    const { commands } = createPalette()
    const others = ['physics/Heat.pdf']

    expect(commands.startCommand('app:note.trash', front({ others }))?.others).toStrictEqual(others)
  })

  it('carries the tab holding it, so the invocation reaches it wherever it went', () => {
    const { commands, openNote } = createPalette()
    openNote('physics/Ontology.md', 'active')

    expect(commands.startCommand('app:note.trash', front())?.note).toBe('active')
  })

  it('leaves the palette on the commands, having nothing to ask', () => {
    const { commands } = createPalette()

    commands.startCommand('app:note.trash', front())

    expect(commands.groups.value.map((group) => group.id)).toStrictEqual([
      'note',
      'window',
      'settings',
      'vault',
    ])
  })
})

describe('destroying a note', () => {
  it('waits for the name of the note, and is not reachable before it', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.delete', front())

    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.chooseItem('exactly', 'exactly')).toBeNull()

    commands.setTyped('Ontolog').catch(rethrow)

    expect(commands.chooseItem('exactly', 'exactly')).toBeNull()
  })

  it('submits once the name of the note is what was typed', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.delete', front())
    commands.setTyped('Ontology').catch(rethrow)

    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(false)
    expect(commands.chooseItem('exactly', 'exactly')?.id).toBe('app:note.delete')
  })
})

/**
 * The vault changes while a step is active. A step targets the note the user
 * named, and the note's path differs from the original.
 */
describe('a note that moves under an open step', () => {
  it('is renamed where it now is, and never at the name it left', () => {
    const { commands, moveNote } = createPalette()
    commands.startCommand('app:note.rename', front())
    moveNote('physics/Ontology.md', 'physics/Being.md', 'Being')
    commands.applyRenames([{ from: 'physics/Ontology.md', to: 'physics/Being.md' }])

    commands.setTyped('Substance').catch(rethrow)

    expect(commands.chooseItem('name', 'name')?.path).toBe('physics/Being.md')
  })

  it('is not destroyed by the name it had, and is by the name it has', () => {
    const { commands, moveNote } = createPalette()
    commands.startCommand('app:note.delete', front())
    moveNote('physics/Ontology.md', 'physics/Being.md', 'Being')
    commands.applyRenames([{ from: 'physics/Ontology.md', to: 'physics/Being.md' }])

    commands.setTyped('Ontology').catch(rethrow)

    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.chooseItem('exactly', 'exactly')).toBeNull()

    commands.setTyped('Being').catch(rethrow)

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Destroy “Being”')
    expect(commands.chooseItem('exactly', 'exactly')?.path).toBe('physics/Being.md')
  })

  it('carries the tab holding it, so the invocation reaches it wherever it went', () => {
    const { commands, openNote } = createPalette()
    openNote('physics/Ontology.md', 'active')
    commands.startCommand('app:note.rename', front())

    commands.setTyped('Substance').catch(rethrow)

    expect(commands.chooseItem('name', 'name')?.note).toBe('active')
  })

  it('leaves a step over another note at its current position', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.rename', front())

    commands.applyRenames([{ from: 'Elsewhere.md', to: 'Moved.md' }])

    commands.setTyped('Substance').catch(rethrow)

    expect(commands.chooseItem('name', 'name')?.path).toBe('physics/Ontology.md')
  })
})

describe('a command that was not offered over what it was requested over', () => {
  it('says the vault could not be opened', () => {
    const { commands } = createPalette({ isReady: false })

    expect(commands.getObjection('app:note.trash', front({ isReady: false }))).toBe(words.noVault)
  })

  it('says nothing in front is a note', () => {
    const { commands } = createPalette()

    expect(
      commands.getObjection('app:note.trash', front({ kind: 'document', path: '', title: '' })),
    ).toBe(words.noNote)
  })

  it('says nothing at all about one that was taken up', () => {
    const { commands } = createPalette()

    expect(commands.getObjection('app:note.trash', front())).toBe('')
    expect(commands.getObjection('nothing of the sort', front())).toBe('')
  })
})

describe('a command that asks for a note', () => {
  it('asks the vault for the names that match, and draws each note once', async () => {
    const { commands, requests } = createPalette({}, [
      name('physics/Entropy.md', 'Entropy'),
      name('physics/Entropy.md', 'Entropy', 'What it counts'),
      name('Order.md', 'Order'),
    ])
    commands.startCommand('app:note.go-to', front())

    await commands.setTyped('en')

    expect(requests).toStrictEqual(['en'])
    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics/Entropy.md',
      'Order.md',
    ])
  })

  it('carries the note that was picked, not the one that was in front', async () => {
    const { commands } = createPalette({}, [name('physics/Entropy.md', 'Entropy')])
    commands.startCommand('app:note.go-to', front())
    await commands.setTyped('en')

    expect(commands.chooseItem('physics/Entropy.md', 'pick')).toStrictEqual({
      id: 'app:note.go-to',
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
    vi.spyOn(console, 'error').mockImplementation(vi.fn())
    const activeTarget = ref(front())
    const commands = useCommandPalette(
      {
        names: async () => Promise.reject(new Error('no model is set')),
        vaults: async () => installation(),
      },
      words,
      () => activeTarget.value,
      { getTitle: () => '', getTabAt: () => null },
      { getStepGroups: () => [], previewItem: vi.fn() },
      runSupport(),
      vi.fn(() => Promise.resolve()),
    )
    commands.setOpen(true)
    commands.startCommand('app:note.go-to', front())

    await commands.setTyped('en')

    expect(commands.groups.value[0]?.silence).toBe(words.noQuery)
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
    'app:appearance.choose-theme': THEMES,
    'app:appearance.choose-mode': HALVES,
  }

  it('offers the groups the window holds, each under its own name', () => {
    const { commands } = createPalette({}, [], target)
    commands.startCommand('app:appearance.choose-theme', front())

    expect(commands.crumb.value).toBe(words.appearance)
    expect(commands.placeholder.value).toBe(words.typeChoice)
    expect(commands.groups.value.map((group) => group.title)).toStrictEqual([
      'Built-in',
      'Custom themes',
    ])
    expect(getItemIds(commands.groups)).toStrictEqual({
      shipping: ['preset:numen', 'preset:dracula'],
      owned: [],
    })
    expect(getEmptyMessage(commands.groups, 'owned')).toBe('A .css file there is one')
  })

  it('offers each command the list it holds for that command', () => {
    const { commands } = createPalette({}, [], target)
    commands.startCommand('app:appearance.choose-mode', front())

    expect(commands.crumb.value).toBe(words.chooseMode)
    expect(getItemIds(commands.groups)).toStrictEqual({
      half: ['mode:system', 'mode:light'],
    })
  })

  it('keeps to the rows the words typed name, and marks where they stand', async () => {
    const { commands } = createPalette({}, [], target)
    commands.startCommand('app:appearance.choose-theme', front())

    await commands.setTyped('rac')

    expect(getItemIds(commands.groups)).toStrictEqual({ shipping: ['preset:dracula'], owned: [] })
    expect(commands.groups.value[0]?.items[0]?.at).toStrictEqual([{ start: 1, end: 4 }])
  })

  it('draws a row the window says cannot be chosen, and chooses nothing by it', () => {
    const { commands } = createPalette({}, [], target)
    commands.startCommand('app:appearance.choose-mode', front())

    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.chooseItem('mode:system', 'chosen')).toBeNull()
    expect(commands.chooseItem('mode:none', 'chosen')).toBeNull()
  })

  it('returns the row that was chosen, from whichever group it stood in', () => {
    const { commands } = createPalette({}, [], target)
    commands.startCommand('app:appearance.choose-theme', front())

    expect(commands.chooseItem('preset:dracula', 'chosen')).toStrictEqual({
      id: 'app:appearance.choose-theme',
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

  it('says which command the keyboard is standing in, and that it remains nowhere after', () => {
    const { commands, displayedLists } = createPalette({}, [], target)
    commands.startCommand('app:appearance.choose-mode', front())

    commands.previewItem('mode:light')
    commands.leaveStep()
    commands.startCommand('app:appearance.choose-theme', front())
    commands.previewItem('preset:dracula')
    commands.leaveStep()

    expect(displayedLists).toStrictEqual([
      'app:appearance.choose-mode mode:light',
      'app:appearance.choose-mode ',
      'app:appearance.choose-theme preset:dracula',
      'app:appearance.choose-theme ',
    ])
  })

  it('says nothing about where the keyboard is at a step that shows nothing', () => {
    const { commands, displayedLists } = createPalette({}, [], target)
    commands.startCommand('app:note.create-child', front())

    commands.previewItem('anything')
    commands.leaveStep()

    expect(displayedLists).toStrictEqual([])
  })
})

describe('leaving a step', () => {
  it('returns to the commands, and the palette stays open', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.create-child', front())

    commands.leaveStep()

    expect(commands.open.value).toBe(true)
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual([
      'note',
      'window',
      'settings',
      'vault',
    ])
  })

  it('puts the commands away from the commands themselves', () => {
    const { commands } = createPalette()

    commands.leaveStep()

    expect(commands.open.value).toBe(false)
  })

  it('passes the field back to the search from the commands themselves', () => {
    const { commands } = createPalette()

    expect(commands.goBack()).toBe(true)
    expect(commands.open.value).toBe(false)
  })

  it('keeps the field on a step, and drops the step', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.create-child', front())

    expect(commands.goBack()).toBe(false)
    expect(commands.crumb.value).toBe(words.command)
  })

  it('is a step of its own to the palette, every time', () => {
    const { commands } = createPalette()
    const steps = [commands.step.value]
    commands.startCommand('app:note.create-child', front())
    steps.push(commands.step.value)
    commands.leaveStep()
    steps.push(commands.step.value)
    commands.startCommand('app:note.rename', front())
    steps.push(commands.step.value)

    expect(new Set(steps).size).toBe(3)
    expect(steps[1]).not.toBe(steps[3])
  })
})

describe('a command requested for from outside the palette', () => {
  it('carries out a command that needs nothing with no name', () => {
    const { commands } = createPalette()

    expect(commands.startCommand('app:note.copy-path', front())?.name).toBe('')
  })

  it('starts from no step where the palette was shut with a step standing', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.rename', front())
    commands.open.value = false

    commands.startCommand('app:note.rename', front())
    commands.leaveStep()

    expect(commands.crumb.value).toBe(words.command)
  })

  it('opens the palette where it asks for what it needs', () => {
    const { commands } = createPalette()
    commands.setOpen(false)

    expect(commands.startCommand('app:note.rename', front())).toBeNull()
    expect(commands.open.value).toBe(true)
    expect(commands.crumb.value).toBe(words.title)
  })

  it('leaves the palette shut where it needs nothing', () => {
    const { commands } = createPalette()
    commands.setOpen(false)

    expect(commands.startCommand('app:note.copy-path', front())?.id).toBe('app:note.copy-path')
    expect(commands.open.value).toBe(false)
  })
})

describe('the commands over the vaults an installation holds', () => {
  /** Only renaming is over the vault in front. The rest ask for one first. */
  it('omits rename command when no vault is active', () => {
    const { commands } = createPalette({ vault: { id: '', name: '' } })

    expect(getItemIds(commands.groups).vault).toStrictEqual([
      'app:note.go-to-start',
      'app:note.go-to',
      'app:vault.open',
      'app:vault.create',
      'app:vault.remove',
      'app:vault.erase',
    ])
  })
})

describe('a command that asks for a vault', () => {
  const two = () => installation(vault('physics', 'Physics'), vault('heat', 'Heat'))

  it('lists the vaults the installation holds, each at the folder it remains in', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.startCommand('app:vault.open', front())

    await settle()

    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics',
      'heat',
    ])
    expect(commands.groups.value[0]?.items[1]?.detail).toBe('/vaults/Heat')
  })

  /** The folder may be on an unmounted volume, and the vault stays listed. */
  it('marks the vault whose folder is not there, and names where it looked', async () => {
    const { commands } = createPalette({}, [], {}, installation(vault('gone', 'Gone', true)))
    commands.startCommand('app:vault.open', front())
    await settle()

    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.groups.value[0]?.items[0]?.detail).toBe(`${words.gone} · /vaults/Gone`)
    expect(commands.chooseItem('gone', 'open')).toBeNull()
  })

  /**
   * The window always relies on something, so the vault in front is not one
   * the list takes. It is drawn with the reason beside it, and choosing it
   * does nothing.
   */
  it('marks active vault and ignores duplicate selection', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.startCommand('app:vault.open', front())
    await settle()

    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.groups.value[0]?.items[0]?.detail).toBe(`${words.current} · /vaults/Physics`)
    expect(commands.chooseItem('physics', 'open')).toBeNull()
  })

  it('says on every row what choosing it asks for', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.startCommand('app:vault.remove', front())
    await settle()

    expect(commands.groups.value[0]?.items[1]?.actions?.[0]?.text).toBe(words.removeVault)
  })

  it('uses selected vault over active vault', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.startCommand('app:vault.open', front())
    await settle()

    expect(commands.chooseItem('heat', 'open')).toMatchObject({
      id: 'app:vault.open',
      vault: { id: 'heat', name: 'Heat' },
    })
  })

  it('keeps the vaults the words typed name, so the field means something', async () => {
    const { commands } = createPalette({}, [], {}, two())
    commands.startCommand('app:vault.open', front())
    await settle()

    await commands.setTyped('hea')

    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual(['heat'])
  })

  it('says the list could not be requested, in the window’s own words', async () => {
    vi.spyOn(console, 'error').mockImplementation(vi.fn())
    const activeTarget = ref(front())
    const commands = useCommandPalette(
      {
        names: async () => [],
        vaults: async () => Promise.reject(new Error('the list is not there')),
      },
      words,
      () => activeTarget.value,
      { getTitle: () => '', getTabAt: () => null },
      { getStepGroups: () => [], previewItem: vi.fn() },
      runSupport(),
      vi.fn(() => Promise.resolve()),
    )
    commands.setOpen(true)

    commands.startCommand('app:vault.open', front())
    await settle()

    expect(commands.groups.value[0]?.silence).toBe(words.noQuery)
  })
})

describe('renaming the vault in front', () => {
  it('remains filled with the name the vault has now', () => {
    const { commands } = createPalette()

    commands.startCommand('app:vault.rename', front())

    expect(commands.crumb.value).toBe(words.renameVault)
    expect(commands.typed.value).toBe('Physics')
  })

  it('passes name to active vault', () => {
    const { commands } = createPalette()
    commands.startCommand('app:vault.rename', front())
    commands.setTyped('Heat').catch(rethrow)

    expect(commands.chooseItem('name', 'name')).toMatchObject({
      id: 'app:vault.rename',
      name: 'Heat',
      vault: { id: 'physics', name: 'Physics' },
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
    commands.startCommand(id, front())
    await settle()
    commands.chooseItem('heat', 'open')
    return commands
  }

  it('is confirmed over the vault that was chosen, saying the folder stays', async () => {
    const commands = await chooseVault('app:vault.remove')

    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual(['no', 'yes'])
    expect(commands.groups.value[0]?.items[0]?.title).toBe(words.keepsVault)
    expect(commands.groups.value[0]?.items[1]?.title).toBe('Remove “Heat”')
    expect(commands.groups.value[0]?.items[1]?.detail).toBe(words.stays)
  })

  it('is carried out over that vault once the question is answered', async () => {
    const commands = await chooseVault('app:vault.remove')

    expect(commands.chooseItem('yes', 'yes')).toMatchObject({
      id: 'app:vault.remove',
      vault: { id: 'heat', name: 'Heat' },
    })
  })

  it('pops the confirmation step and returns null when answered with no', async () => {
    const commands = await chooseVault('app:vault.remove')

    expect(commands.chooseItem('no', 'no')).toBeNull()
    expect(commands.groups.value[0]?.id).toBe('vaults')
  })

  it('returns the list, with the vaults on it, where the step is left', async () => {
    const commands = await chooseVault('app:vault.remove')

    commands.leaveStep()
    await settle()

    expect(commands.groups.value[0]?.id).toBe('vaults')
    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual([
      'physics',
      'heat',
    ])
  })

  it('waits for the name of the vault chosen, and says where its folder is located', async () => {
    const commands = await chooseVault('app:vault.erase')

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Erase “Heat”')
    expect(commands.groups.value[0]?.items[0]?.detail).toBe(words.binned)
    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(true)
    expect(commands.placeholder.value).toBe(words.typeVaultBack)
    expect(commands.chooseItem('exactly', 'exactly')).toBeNull()
  })

  it('submits once the name of that vault is what was typed', async () => {
    const commands = await chooseVault('app:vault.erase')

    commands.setTyped('Heat').catch(rethrow)

    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(false)
    expect(commands.chooseItem('exactly', 'exactly')).toMatchObject({
      id: 'app:vault.erase',
      vault: { id: 'heat', name: 'Heat' },
    })
  })

  it('ignores active vault name', async () => {
    const commands = await chooseVault('app:vault.erase')

    commands.setTyped('Physics').catch(rethrow)

    expect(commands.chooseItem('exactly', 'exactly')).toBeNull()
  })
})

/**
 * The commands over a vault that responds when the test resolves it, and a
 * pending action that ends when the test resolves it. Nothing is opened.
 */
const createSlowPalette = (offers: Record<string, readonly StepGroup[]> = {}) => {
  const names: Deferred<readonly NameMatch[]>[] = []
  const lists: Deferred<VaultList>[] = []
  const holds: Deferred<void>[] = []
  const request: string[] = []
  const core = {
    names: (query: string) => {
      request.push(query)
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
  const commands = useCommandPalette(
    core,
    words,
    () => front(),
    createLookup().knows,
    createLists(offers).paletteLists,
    runSupport(),
    wait,
  )
  return { commands, names, lists, holds, hold, requests: request }
}

/** The state of the one group a step renders. */
const describeGroup = (groups: ReturnType<typeof createPalette>['commands']['groups']) => {
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

    expect(commands.typed.value).toBe('')
  })
})

describe('a step that asks for a note, as the vault answers', () => {
  it('opens working at nothing and asking for a note', () => {
    const { commands } = createSlowPalette()

    commands.startCommand('app:note.go-to', front())

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.typeNote,
    })
  })

  it('opens saying nothing of a list a step before it could not ask', async () => {
    const { commands, lists } = createSlowPalette()
    commands.startCommand('app:vault.open', front())
    lists[0]?.fail('the list is not there')
    await settle()
    commands.leaveStep()

    commands.startCommand('app:note.go-to', front())

    expect(getEmptyMessage(commands.groups, 'picking')).toBe(words.typeNote)
  })

  it('is working while the vault has not answered', async () => {
    const { commands } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())

    commands.setTyped('en').catch(rethrow)
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: [],
      isWorking: true,
      silence: words.noneFound,
    })
  })

  it('stops working once the vault answers', async () => {
    const { commands, names } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('en').catch(rethrow)
    await settle()

    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: ['physics/Entropy.md'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('asks about the words and not the spaces around them', async () => {
    const { commands, requests } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())

    commands.setTyped('  en  ').catch(rethrow)
    await settle()

    expect(requests).toStrictEqual(['en'])
  })

  it('asks nothing once what was typed is taken back, and draws nothing found', async () => {
    const { commands, names, requests } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('en').catch(rethrow)
    await settle()
    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    commands.setTyped('').catch(rethrow)
    await settle()

    expect(requests).toStrictEqual(['en'])
    expect(describeGroup(commands.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.typeNote,
    })
  })

  it('asks nothing about a keystroke another one overtook while it waited', async () => {
    const { commands, holds, hold, requests } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    hold.isOn = true

    commands.setTyped('e').catch(rethrow)
    commands.setTyped('en').catch(rethrow)
    holds[0]?.answer()
    await settle()

    expect(requests).toStrictEqual([])

    holds[1]?.answer()
    await settle()

    expect(requests).toStrictEqual(['en'])
  })

  it('draws the answer to the question requested last, whatever lands after it', async () => {
    const { commands, names } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('e').catch(rethrow)
    await settle()
    commands.setTyped('en').catch(rethrow)
    await settle()

    names[1]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()
    names[0]?.answer([name('Ether.md', 'Ether')])
    await settle()

    expect(describeGroup(commands.groups).items).toStrictEqual(['physics/Entropy.md'])
  })

  it('draws no failure of a question nobody is asking any more', async () => {
    const { commands, names } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('e').catch(rethrow)
    await settle()
    commands.setTyped('en').catch(rethrow)
    await settle()

    names[1]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()
    names[0]?.fail('the vault went away')
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: ['physics/Entropy.md'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('keeps working where only a question nobody is asking has answered', async () => {
    const { commands, names } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('e').catch(rethrow)
    await settle()
    commands.setTyped('en').catch(rethrow)
    await settle()

    names[0]?.answer([name('Ether.md', 'Ether')])
    await settle()

    expect(describeGroup(commands.groups).isWorking).toBe(true)
  })

  it('holds nothing where the vault could not answer', async () => {
    const { commands, names } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('en').catch(rethrow)
    await settle()

    names[0]?.fail('no model is set')
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.noQuery,
    })
  })

  it('says nothing of a failure while the next question is out', async () => {
    const { commands, names } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('e').catch(rethrow)
    await settle()
    names[0]?.fail('no model is set')
    await settle()

    commands.setTyped('en').catch(rethrow)
    await settle()

    expect(getEmptyMessage(commands.groups, 'picking')).toBe(words.noneFound)
  })

  it('draws no answer to a question requested at a step that was left', async () => {
    const { commands, names } = createSlowPalette()
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('en').catch(rethrow)
    await settle()
    commands.leaveStep()
    commands.startCommand('app:note.go-to', front())

    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    expect(describeGroup(commands.groups).items).toStrictEqual([])
  })

  it('knows no note of an answer that lands after the palette is put away', async () => {
    const { commands, names } = createSlowPalette()
    commands.setOpen(true)
    commands.startCommand('app:note.go-to', front())
    commands.setTyped('en').catch(rethrow)
    await settle()

    commands.setOpen(false)
    names[0]?.answer([name('physics/Entropy.md', 'Entropy')])
    await settle()

    expect(commands.findNoteType('physics/Entropy.md')).toBeNull()
  })

  it('asks the vault nothing at a step that does not pick a note', async () => {
    const { commands, requests } = createSlowPalette()
    commands.startCommand('app:note.create-child', front())

    commands.setTyped('Entropy').catch(rethrow)
    await settle()

    expect(requests).toStrictEqual([])
  })
})

describe('a step that asks for a vault, as the list answers', () => {
  const two = () => installation(vault('physics', 'Physics'), vault('heat', 'Heat'))

  it('holds nothing and is working while the list has not come back', async () => {
    const { commands } = createSlowPalette()

    commands.startCommand('app:vault.open', front())
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: [],
      isWorking: true,
      silence: words.noneFound,
    })
  })

  it('draws the list requested for last, and is working until that one is back', async () => {
    const { commands, lists } = createSlowPalette()
    commands.startCommand('app:vault.open', front())
    commands.leaveStep()
    commands.startCommand('app:vault.open', front())

    lists[0]?.answer(two())
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: [],
      isWorking: true,
      silence: words.noneFound,
    })

    lists[1]?.answer(installation(vault('heat', 'Heat')))
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: ['heat'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('draws no failure of a list nobody is asking for any more', async () => {
    const { commands, lists } = createSlowPalette()
    commands.startCommand('app:vault.open', front())
    commands.leaveStep()
    commands.startCommand('app:vault.open', front())

    lists[1]?.answer(two())
    await settle()
    lists[0]?.fail('the list is not there')
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: ['physics', 'heat'],
      isWorking: false,
      silence: words.noneFound,
    })
  })

  it('holds nothing where the list could not be requested', async () => {
    const { commands, lists } = createSlowPalette()
    commands.startCommand('app:vault.open', front())

    lists[0]?.fail('the list is not there')
    await settle()

    expect(describeGroup(commands.groups)).toStrictEqual({
      items: [],
      isWorking: false,
      silence: words.noQuery,
    })
  })

  it('keeps the vaults the words typed name, around the spaces typed', async () => {
    const { commands, lists } = createSlowPalette()
    commands.startCommand('app:vault.open', front())
    lists[0]?.answer(two())
    await settle()

    await commands.setTyped('  hea  ')

    expect(describeGroup(commands.groups).items).toStrictEqual(['heat'])
  })
})

describe('a command requested for while a step remains open', () => {
  it('remains over that step, and leaving it returns to it', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.create-child', front())

    commands.startCommand('app:note.rename', front())
    commands.leaveStep()

    expect(commands.crumb.value).toBe(words.child)
  })
})

describe('what a step draws', () => {
  it('lists each found note by title, highlights name matches and falls back to the file name', async () => {
    const { commands } = createPalette({}, [
      name('physics/Entropy.md', 'Entropy'),
      name('physics/Heat.md', 'Heat', 'What it counts'),
      name('Untitled.md', ''),
    ])
    commands.startCommand('app:note.go-to', front())

    await commands.setTyped('en')

    expect(commands.groups.value[0]?.id).toBe('picking')
    expect(commands.groups.value[0]?.items).toStrictEqual([
      {
        id: 'physics/Entropy.md',
        title: 'Entropy',
        at: [{ start: 0, end: 1 }],
        actions: [{ id: 'pick', text: words.travel }],
      },
      {
        id: 'physics/Heat.md',
        title: 'Heat',
        actions: [{ id: 'pick', text: words.travel }],
      },
      {
        id: 'Untitled.md',
        title: 'Untitled.md',
        at: [{ start: 0, end: 1 }],
        actions: [{ id: 'pick', text: words.travel }],
      },
    ])
  })

  it('says which of four each note found is, and nothing of a note it did not find', async () => {
    const { commands } = createPalette({}, [
      name('physics/Entropy.md', 'Entropy', '', 'sample'),
      name('physics/Heat.md', 'Heat', '', 'form'),
    ])
    commands.startCommand('app:note.go-to', front())

    await commands.setTyped('en')

    expect(commands.findNoteType('physics/Heat.md')).toBe('form')
    expect(commands.findNoteType('physics/Entropy.md')).toBe('sample')
    expect(commands.findNoteType('Order.md')).toBeNull()
  })

  it('asks for a note where only spaces are typed', async () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.go-to', front())

    await commands.setTyped('   ')

    expect(getEmptyMessage(commands.groups, 'picking')).toBe(words.typeNote)
  })

  it('confirms under the answer each row gives, filtered by the words typed around spaces', async () => {
    const { commands } = createPalette(
      {},
      [],
      {},
      installation(vault('physics', 'Physics'), vault('heat', 'Heat')),
    )
    commands.startCommand('app:vault.remove', front())
    await settle()
    commands.chooseItem('heat', 'open')

    expect(commands.groups.value[0]?.items.map((item) => item.actions)).toStrictEqual([
      [{ id: 'no', text: words.keepsVault }],
      [{ id: 'yes', text: words.removeVaultAction }],
    ])

    commands.setTyped('  remo  ').catch(rethrow)

    expect(commands.groups.value[0]?.items.map((item) => item.id)).toStrictEqual(['yes'])
  })

  it('takes the name typed back around spaces, under the one action that destroys', () => {
    const { commands } = createPalette()
    commands.startCommand('app:note.delete', front())

    commands.setTyped('  Ontology  ').catch(rethrow)

    expect(commands.groups.value[0]?.id).toBe('exactly')
    expect(commands.groups.value[0]?.items[0]?.isDisabled).toBe(false)
    expect(commands.groups.value[0]?.items[0]?.actions).toStrictEqual([
      { id: 'exactly', text: words.destroys },
    ])
    expect(commands.chooseItem('exactly', 'exactly')?.name).toBe('Ontology')
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
    const { commands } = createPalette({}, [], { 'app:appearance.choose-theme': THEMES })
    commands.startCommand('app:appearance.choose-theme', front())

    expect(commands.groups.value[0]?.items).toStrictEqual([
      {
        id: 'preset:numen',
        title: 'numen',
        detail: 'Current',
        actions: [{ id: 'chosen', text: words.chooses }],
      },
      {
        id: 'preset:dracula',
        title: 'dracula',
        actions: [{ id: 'chosen', text: words.chooses }],
      },
    ])
  })

  it('is lit from its first letter where the words typed begin it', async () => {
    const { commands } = createPalette({}, [], { 'app:appearance.choose-theme': THEMES })
    commands.startCommand('app:appearance.choose-theme', front())

    await commands.setTyped('num')

    expect(commands.groups.value[0]?.items.map((item) => item.at)).toStrictEqual([
      [{ start: 0, end: 3 }],
    ])
  })

  it('is kept by the words typed around spaces', async () => {
    const { commands } = createPalette({}, [], { 'app:appearance.choose-theme': THEMES })
    commands.startCommand('app:appearance.choose-theme', front())

    await commands.setTyped('  rac  ')

    expect(commands.groups.value[0]?.items.map((item) => item.at)).toStrictEqual([
      [{ start: 1, end: 4 }],
    ])
  })
})

describe('the icon of a command', () => {
  it('is the one its declaration gives, and none for a command that gives none', () => {
    const icon = defineComponent({ name: 'GraphIcon', render: () => null })
    const commands = useCommandPalette(
      { names: async () => [], vaults: async () => installation(vault('physics', 'Physics')) },
      words,
      () => front(),
      createLookup().knows,
      createLists({}).paletteLists,
      runSupport(),
      vi.fn(() => Promise.resolve()),
      () => [
        {
          id: 'org.example.sample:graph.run',
          text: 'New graph',
          icon,
          group: 'window',
          isAvailable: () => true,
        },
      ],
    )

    expect(commands.findCommandIcon('org.example.sample:graph.run')).toBe(icon)
    expect(commands.findCommandIcon('app:note.open')).toBeNull()
  })
})

describe('runSupport, arguments steps, and palette shortcuts', () => {
  it('tracks unsupported runs in runSupport', () => {
    const runs = runSupport()
    expect(runs.canRun('ocr')).toBe(true)
    runs.cannotRun('ocr')
    expect(runs.canRun('ocr')).toBe(false)
    expect(runs.canRun('other')).toBe(true)
  })

  it('walks through multi-prompt arguments steps and handles commands without prompts', () => {
    const commands = useCommandPalette({
      core: {
        names: async () => [],
        vaults: async () => installation(vault('physics', 'Physics')),
      },
      words,
      at: () => front(),
      knows: createLookup().knows,
      lists: createLists({}).paletteLists,
      runs: runSupport(),
      getExtraCommands: () => [
        {
          id: 'org.example.sample:args.run',
          text: 'With args',
          group: 'window',
          needs: 'arguments',
          prompts: [
            { name: 'topic', title: 'Topic title', isRequired: true },
            { name: 'detail', title: 'Detail title', isRequired: false },
          ],
          isAvailable: () => true,
        },
        {
          id: 'org.example.sample:no-prompts.run',
          text: 'No prompts',
          group: 'window',
          needs: 'arguments',
          isAvailable: () => true,
        },
      ],
    })

    commands.startCommand('org.example.sample:no-prompts.run', front())
    expect(commands.placeholder.value).toBe(words.typeCommand)
    expect(commands.chooseItem('name', 'name')).toBeNull()
    commands.leaveStep()

    commands.startCommand('org.example.sample:args.run', front())
    expect(commands.placeholder.value).toBe('Topic title')
    expect(commands.chooseItem('name', 'name')).toBeNull()

    commands.setTyped('  Quantum  ').catch(rethrow)
    expect(commands.chooseItem('name', 'name')).toBeNull()
    expect(commands.placeholder.value).toBe('Detail title')

    const result = commands.chooseItem('name', 'name')
    expect(result?.id).toBe('org.example.sample:args.run')
    expect(result?.values).toStrictEqual({ topic: 'Quantum' })
  })

  it('toggles search and command palettes via keyboard shortcuts', () => {
    const searchOpen = ref(false)
    const search = {
      open: searchOpen,
      setOpen: (val: boolean) => {
        searchOpen.value = val
      },
    } as unknown as SearchState
    const { commands } = createPalette()
    commands.setOpen(false)

    const Comp = defineComponent({
      setup() {
        usePaletteShortcuts(search, commands)
        return () => h('div')
      },
    })
    const wrapper = mount(Comp)

    const nonChord = new KeyboardEvent('keydown', { key: 'k', cancelable: true })
    globalThis.dispatchEvent(nonChord)
    expect(searchOpen.value).toBe(false)

    const prePrevented = new KeyboardEvent('keydown', {
      key: 'k',
      ctrlKey: true,
      cancelable: true,
    })
    prePrevented.preventDefault()
    globalThis.dispatchEvent(prePrevented)
    expect(searchOpen.value).toBe(false)

    const kEvent = new KeyboardEvent('keydown', { key: 'k', ctrlKey: true, cancelable: true })
    globalThis.dispatchEvent(kEvent)
    expect(kEvent.defaultPrevented).toBe(true)
    expect(searchOpen.value).toBe(true)
    expect(commands.open.value).toBe(false)

    const pEvent = new KeyboardEvent('keydown', { key: 'p', ctrlKey: true, cancelable: true })
    globalThis.dispatchEvent(pEvent)
    expect(pEvent.defaultPrevented).toBe(true)
    expect(searchOpen.value).toBe(false)
    expect(commands.open.value).toBe(true)

    wrapper.unmount()
  })
})
