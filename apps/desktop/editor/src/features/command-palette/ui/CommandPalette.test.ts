import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { Palette } from '@numen/shell'
import type { SlotCommand } from '@numen/plugin-host'
import { WORDS as words } from '@/shared/words'
import { runSupport } from '../model/palette/runSupport'
import { useCommandPalette } from '../model/palette/palette'
import { useSearch, type NameMatch, type Passage, type SearchDeps } from '../model/search/search'
import type { CommandDeps } from '../model/handlers/deps'
import type { CommandTarget } from '../types'
import CommandPalette from './CommandPalette.vue'
import type * as HandlersModule from '../model/handlers/handlers'

const h = vi.hoisted(() => ({ runInvocation: vi.fn(() => Promise.resolve()) }))

vi.mock('../model/handlers/handlers', async (importOriginal) => ({
  ...(await importOriginal<typeof HandlersModule>()),
  runInvocation: h.runInvocation,
}))

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

async function mountField(found: { names?: NameMatch[]; passages?: Passage[] }, text: string) {
  const core: SearchDeps = {
    names: async () => found.names ?? [],
    search: async (_query, mode) => (mode === 'words' ? (found.passages ?? []) : []),
  }
  const search = useSearch(core, words, {
    wait: now,
    getTarget: () => inFront,
    menu: { getEntries: () => [SHOW] },
  })
  const commands = useCommandPalette(
    { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
    words,
    () => inFront,
    { getTitle: () => '', getTabAt: () => null },
    { getStepGroups: () => [], previewItem: () => undefined },
    runSupport(),
  )
  search.setOpen(true)
  await search.setTyped(text)
  await flushPromises()
  const runChoice = vi.fn()
  const field = mount(CommandPalette, {
    props: {
      commands,
      search,
      doing: {} as CommandDeps,
      getTarget: () => inFront,
      places: {},
      runChoice,
    },
    global: { stubs: { Palette: true } },
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

beforeEach(() => {
  h.runInvocation.mockClear()
})

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
    expect(h.runInvocation).not.toHaveBeenCalled()
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
    expect(h.runInvocation).not.toHaveBeenCalled()
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
      expect(h.runInvocation).not.toHaveBeenCalled()
      expect(one.search.open.value).toBe(false)
    },
  )
})

describe('choosing a row of the commands', () => {
  it('runs the invocation of the command with the deps of the window and no choice of a result', async () => {
    const one = await mountField({}, 'x')
    one.search.setOpen(false)
    one.commands.setOpen(true)

    await one.choose('app:file.copyPath', 'app:file.copyPath')

    expect(h.runInvocation).toHaveBeenCalledTimes(1)
    expect(one.runChoice).not.toHaveBeenCalled()
  })
})

function createDeferred<T>() {
  let answer = (_value: T) => undefined as void
  const promise = new Promise<T>((resolve) => {
    answer = resolve
  })
  return { promise, answer }
}

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
    const commands = useCommandPalette(
      { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
      words,
      () => inFront,
      { getTitle: () => '', getTabAt: () => null },
      { getStepGroups: () => [], previewItem: () => undefined },
      runSupport(),
    )
    const field = mount(CommandPalette, {
      props: {
        commands,
        search,
        doing: {} as CommandDeps,
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
