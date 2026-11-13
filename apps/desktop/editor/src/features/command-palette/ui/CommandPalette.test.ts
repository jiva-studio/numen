import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { Palette } from '@numen/workbench'
import type { SlotCommand } from '@numen/plugin-loader'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../words'
import { createAppCommands } from '@/testing/appCommands'
import { runSupport } from '../model/palette/runSupport'
import { createCommandPalette } from '../model/palette/palette'
import { useSearch, type NameMatch, type Passage, type SearchDeps } from '../model/search/search'
import type { CommandDeps } from '@/shared/model/command'
import { createCommandRegistry } from '@/shared/model/command'
import type { CommandTarget } from '@/shared/model/command'
import CommandPalette from './CommandPalette.vue'
import { createDeferred } from '@/testing/deferred'

const words = { ...windowWords, ...PALETTE_WORDS }

const COPY_PATH = 'app:file.copyPath'
const copyPath = vi.fn()

let registry = createCommandRegistry()

beforeEach(() => {
  copyPath.mockClear()
  registry = createCommandRegistry()
  registry.register(COPY_PATH, copyPath)
})

const ATLAS = 'org.example.atlas'
const SHOW: SlotCommand = {
  kind: 'command',
  key: `${ATLAS}/command/graph.showNote`,
  pluginId: ATLAS,
  command: `${ATLAS}:graph.showNote`,
  title: 'Show note in plex',
  shortcut: '',
  hasItems: false,
}

const inFront: CommandTarget = {
  tab: 'tab',
  kind: 'note',
  path: 'physics/Being.md',
  title: 'Being',
  file: 'physics/Being.md',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
}

const heading: NameMatch = {
  path: 'reef/Coral.md',
  title: 'Coral',
  heading: 'Polyps',
  line: 7,
  at: [],
  type: 'note',
}

const byName: NameMatch = { ...heading, heading: '', line: -1 }

const notePassage: Passage = {
  path: 'reef/Coral.md',
  title: 'Coral',
  isNote: true,
  type: 'note',
  selector: null,
  text: 'polyps build the reef',
  start: 12,
  length: 4,
  line: 3,
  at: [],
}

const bookPassage: Passage = {
  ...notePassage,
  path: 'books/Atlas.pdf',
  title: '',
  isNote: false,
  selector: { page: 3 },
}

const now = () => Promise.resolve()
const flushPromises = async () => {
  for (let turn = 0; turn < 8; turn += 1) await Promise.resolve()
}

async function mountField(
  found: { names?: NameMatch[]; passages?: Passage[] },
  text: string,
  showSilence = false,
) {
  const core: SearchDeps = {
    names: async () => found.names ?? [],
    search: async (_query, mode) => (mode === 'words' ? (found.passages ?? []) : []),
  }
  const search = useSearch(core, words, {
    wait: now,
    getTarget: () => inFront,
    menu: { getEntries: () => [SHOW] },
  })
  const commands = createCommandPalette({
    core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
    words,
    at: () => inFront,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => [], previewItem: () => undefined },
    runs: runSupport(),
    getCommands: createAppCommands,
  })
  search.setOpen(true)
  await search.setTyped(text)
  await flushPromises()
  const runChoice = vi.fn()
  const field = mount(CommandPalette, {
    props: {
      commands,
      search,
      doing: { writeMessage: vi.fn(), registry } as unknown as CommandDeps,
      getTarget: () => inFront,
      places: {},
      runChoice,
    },
    global: {
      stubs: { Palette: showSilence ? { template: '<div><slot name="silence" /></div>' } : true },
    },
  })
  const choose = async (item: string, action: string) => {
    field.findComponent(Palette).vm.$emit('choose', item, action)
    await flushPromises()
  }
  const rows = search.groups.value.flatMap((group) =>
    group.items.map((one) => ({ item: one.id, actions: one.actions?.map((a) => a.id) ?? [] })),
  )
  return { search, commands, runChoice, choose, rows, field }
}

describe('choosing an action of a row of the search', () => {
  it('runs the choice the search answers for every action of every row, and nothing else', async () => {
    const found = { names: [byName, heading], passages: [notePassage, bookPassage] }
    const choices = (await mountField(found, 'coral')).rows.flatMap((row) =>
      row.actions.map((action) => [row.item, action] as const),
    )
    expect(choices.length).toBeGreaterThanOrEqual(6)

    for (const [item, action] of choices) {
      const one = await mountField(found, 'coral')
      const expected = one.search.chooseResult(item, action)
      expect(expected, `${item} ${action}`).not.toBeNull()

      await one.choose(item, action)

      expect(one.runChoice, `${item} ${action}`).toHaveBeenCalledExactlyOnceWith(expected)
    }
    expect(copyPath).not.toHaveBeenCalled()
  })

  it('runs the command that opens a document over its file for the action of a document', async () => {
    const one = await mountField({ passages: [bookPassage] }, 'reef')
    const row = one.rows.find((each) => each.item.includes('books/Atlas.pdf'))
    expect(row).toBeDefined()

    await one.choose(row!.item, 'app:file.open#document')

    expect(one.runChoice).toHaveBeenCalledExactlyOnceWith({
      command: 'app:file.open#document',
      over: { path: '', file: 'books/Atlas.pdf', title: '', selector: { page: 3 } },
      input: '',
    })
  })

  it('closes the search once and runs the choice once', async () => {
    const one = await mountField({ passages: [bookPassage] }, 'reef')
    const row = one.rows.find((each) => each.item.includes('books/Atlas.pdf'))
    expect(row).toBeDefined()
    expect(one.search.open.value).toBe(true)

    await one.choose(row!.item, 'app:file.open#document')

    expect(one.search.open.value).toBe(false)
    expect(one.runChoice).toHaveBeenCalledTimes(1)
  })

  it('runs nothing for an action the row does not carry', async () => {
    const one = await mountField({ passages: [bookPassage] }, 'reef')
    const row = one.rows.find((each) => each.item.includes('books/Atlas.pdf'))
    expect(row).toBeDefined()

    await one.choose(row!.item, 'note')

    expect(one.runChoice).not.toHaveBeenCalled()
    expect(copyPath).not.toHaveBeenCalled()
  })
})

describe('choosing an action that creates a note on a row of the search', () => {
  it.each([
    'app:note.create',
    'app:note.createChild',
    'app:note.createParent',
    'app:note.createJump',
  ])(
    'runs %s with the words typed as its input, as a command and not as a branch',
    async (action) => {
      const one = await mountField({ names: [byName] }, '  Entropy  ')

      await one.choose(byName.path, action)

      expect(one.runChoice).toHaveBeenCalledExactlyOnceWith({
        command: action,
        over: null,
        input: 'Entropy',
      })
      expect(copyPath).not.toHaveBeenCalled()
      expect(one.search.open.value).toBe(false)
    },
  )
})

describe('choosing a row of the commands', () => {
  it('runs the invocation of the command with the deps of the window and no choice of a result', async () => {
    const one = await mountField({}, 'x')
    one.search.setOpen(false)
    one.commands.palette.setOpen(true)

    await one.choose(COPY_PATH, COPY_PATH)

    expect(copyPath).toHaveBeenCalledTimes(1)
    expect(one.runChoice).not.toHaveBeenCalled()
  })
})

describe('the highlight of the search menu as groups answer', () => {
  it('keeps the highlight on the selected row when another search group answers', async () => {
    const names = [createDeferred<NameMatch[]>(), createDeferred<NameMatch[]>()]
    const wordsGroup = [createDeferred<Passage[]>(), createDeferred<Passage[]>()]
    const meaningGroup = [createDeferred<Passage[]>(), createDeferred<Passage[]>()]
    const holds = [createDeferred<void>(), createDeferred<void>()]
    let namesTurn = 0
    let wordsTurn = 0
    let meaningTurn = 0
    let holdTurn = 0
    const core: SearchDeps = {
      names: () => names[namesTurn++]!.promise,
      search: (_query, mode) =>
        mode === 'words' ? wordsGroup[wordsTurn++]!.promise : meaningGroup[meaningTurn++]!.promise,
    }
    const search = useSearch(core, words, {
      wait: () => holds[holdTurn++]!.promise,
      getTarget: () => inFront,
      menu: { getEntries: () => [SHOW] },
    })
    const commands = createCommandPalette({
      core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
      words,
      at: () => inFront,
      knows: { getTitle: () => '', getTabAt: () => null },
      lists: { getStepGroups: () => [], previewItem: () => undefined },
      runs: runSupport(),
    })
    const field = mount(CommandPalette, {
      props: {
        commands,
        search,
        doing: { registry } as unknown as CommandDeps,
        getTarget: () => inFront,
        places: {},
        runChoice: vi.fn(),
      },
      global: { stubs: { teleport: true } },
    })

    try {
      search.setOpen(true)
      await flushPromises()

      await field.get('input[type="text"]').setValue('coral')
      await flushPromises()
      expect(field.findAll('[role="option"]')).toHaveLength(0)

      holds[0]!.answer()
      await flushPromises()
      names[0]!.answer([byName, heading])
      wordsGroup[0]!.answer([notePassage])
      meaningGroup[0]!.answer([bookPassage])
      await flushPromises()

      await field.get('input[type="text"]').setValue('entropy')
      await flushPromises()
      expect(field.findAll('[role="option"]')).toHaveLength(0)

      holds[1]!.answer()
      await flushPromises()

      names[1]!.answer([
        { ...byName, path: 'notes/Entropy.md', title: 'Entropy' },
        { ...byName, path: 'notes/Thermodynamics.md', title: 'Thermodynamics' },
      ])
      await flushPromises()

      const input = field.get('input[type="text"]')
      const selectedTitle = () =>
        field.get('[role="option"][aria-selected="true"] [data-palette="name"]').text()

      expect(selectedTitle()).toBe('Entropy')

      await input.trigger('keydown', { key: 'ArrowDown' })
      await flushPromises()

      expect(selectedTitle()).toBe('Thermodynamics')

      wordsGroup[1]!.answer([notePassage])
      await flushPromises()

      expect(selectedTitle()).toBe('Thermodynamics')

      meaningGroup[1]!.answer([bookPassage])
      await flushPromises()

      expect(selectedTitle()).toBe('Thermodynamics')
    } finally {
      field.unmount()
    }
  })
})

describe('the empty state of the search', () => {
  it('shows the typed words and the icon when nothing is found', async () => {
    const one = await mountField({}, '  oo  ', true)

    expect(one.field.get('[data-palette="nothing-text"]').text()).toBe('No results for “oo”')
    expect(one.field.find('[data-palette="nothing-icon"]').exists()).toBe(true)
  })

  it('is absent while results exist', async () => {
    const one = await mountField({ passages: [bookPassage] }, 'reef', true)

    expect(one.field.find('[data-palette="nothing-text"]').exists()).toBe(false)
    expect(one.field.find('[data-palette="nothing-icon"]').exists()).toBe(false)
  })
})

describe('the spinner of the search field', () => {
  it('turns while the meaning of the query is out and stops once it answers', async () => {
    const meaning = createDeferred<readonly Passage[]>()
    const core: SearchDeps = {
      names: async () => [],
      search: async (_query, mode) => (mode === 'meaning' ? meaning.promise : [notePassage]),
    }
    const search = useSearch(core, words, { wait: now })
    const commands = createCommandPalette({
      core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
      words,
      at: () => inFront,
      knows: { getTitle: () => '', getTabAt: () => null },
      lists: { getStepGroups: () => [], previewItem: () => undefined },
      runs: runSupport(),
      getCommands: createAppCommands,
    })
    const field = mount(CommandPalette, {
      props: {
        commands,
        search,
        doing: { writeMessage: vi.fn(), registry } as unknown as CommandDeps,
        getTarget: () => inFront,
        places: {},
        runChoice: vi.fn(),
      },
      global: { stubs: { Palette: true } },
    })
    search.setOpen(true)
    const typed = search.setTyped('reef')
    await flushPromises()

    expect(field.findComponent(Palette).props('isBusy')).toBe(true)

    meaning.answer([])
    await typed
    await flushPromises()

    expect(field.findComponent(Palette).props('isBusy')).toBe(false)
  })
})
