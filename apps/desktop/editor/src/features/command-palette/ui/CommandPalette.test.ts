/** What the field of the window does with the row and the action a person chooses. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { Palette } from '@numen/shell'
import type { PlacedCommand } from '@numen/plugin-host'
import { WORDS as words } from '@/shared/words'
import { CREATE_NOTE_ROW } from '../lib/offers/offers'
import { runSupport } from '../model/palette/runSupport'
import { useCommandPalette } from '../model/palette/palette'
import { useSearch, type NameMatch, type Passage, type SearchDeps } from '../model/search/search'
import type { CommandDeps } from '../model/handlers/deps'
import type { CommandTarget } from '../types'
import CommandPalette from './CommandPalette.vue'

const h = vi.hoisted(() => ({ runInvocation: vi.fn(() => Promise.resolve()) }))

vi.mock('../model/handlers/handlers', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../model/handlers/handlers')>()),
  runInvocation: h.runInvocation,
}))

const ATLAS = 'org.example.atlas'
const SHOW: PlacedCommand = {
  pluginId: ATLAS,
  command: `${ATLAS}:graph.showNote`,
  title: 'Show note in plex',
  group: '',
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

/** The window's field, with a search that answered with the names and passages given. */
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

    await one.choose(row!.item, 'app:file.open')

    expect(one.runChoice).toHaveBeenCalledExactlyOnceWith({
      command: 'app:file.open',
      over: { path: '', file: 'books/Atlas.pdf', title: '', selector: { page: 3 } },
      input: '',
    })
  })

  it('closes the search once and runs the choice once', async () => {
    const one = await mountField({ passages: [bookPassage] }, 'reef')
    const row = one.rows.find((each) => each.item.includes('books/Atlas.pdf'))
    expect(row).toBeDefined()
    expect(one.search.open.value).toBe(true)

    await one.choose(row!.item, 'app:file.open')

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

describe('choosing an action of the row that creates a note', () => {
  it.each([
    'app:note.create',
    'app:note.createChild',
    'app:note.createParent',
    'app:note.createJump',
  ])(
    'runs %s with the words typed as its input, as a command and not as a branch',
    async (action) => {
      const one = await mountField({}, '  Entropy  ')

      await one.choose(CREATE_NOTE_ROW, action)

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

    await one.choose('app:note.copyPath', 'app:note.copyPath')

    expect(h.runInvocation).toHaveBeenCalledTimes(1)
    expect(one.runChoice).not.toHaveBeenCalled()
  })
})
