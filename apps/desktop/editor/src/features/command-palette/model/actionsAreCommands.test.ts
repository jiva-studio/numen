/**
 * Every action the palette draws on a row is the id of a command.
 */
import { describe, expect, it } from 'vitest'
import type { PaletteGroup } from '@numen/workbench'
import type { SlotCommand } from '@numen/plugin-loader'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../words'
import { createAppCommands } from '@/testing/appCommands'
import { runSupport } from './palette/runSupport'
import { createCommandPalette } from './palette/palette'
import { useSearch, type NameMatch, type SearchDeps } from './search/search'
import type { SearchResult } from './search/lookup'
import type { CommandTarget } from '@/shared/model/command'

const words = { ...windowWords, ...PALETTE_WORDS }

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
const nothingInFront: CommandTarget = {
  ...inFront,
  tab: '',
  kind: null,
  path: '',
  title: '',
  file: '',
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
const noteResult: SearchResult = {
  path: 'reef/Coral.md',
  title: 'Coral',
  isNote: true,
  type: 'note',
  selector: null,
  text: 'polyps build the reef',
  start: 12,
  length: 4,
  matchLine: 3,
  matchRanges: [],
}
const bookResult: SearchResult = {
  ...noteResult,
  path: 'books/Atlas.pdf',
  title: '',
  isNote: false,
  selector: { page: 3 },
}

const now = () => Promise.resolve()
const flushPromises = async () => {
  for (let turn = 0; turn < 8; turn += 1) await Promise.resolve()
}

const collectActionIds = (groups: readonly PaletteGroup[]): string[] =>
  groups.flatMap((group) =>
    group.items.flatMap((one) => (one.actions ?? [{ id: one.id }]).map((action) => action.id)),
  )

const searchFor = async (
  text: string,
  found: { names?: NameMatch[]; searchResults?: SearchResult[] },
  target: CommandTarget,
) => {
  const core: SearchDeps = {
    searchNames: async () => found.names ?? [],
    search: async (_query, mode) => (mode === 'words' ? (found.searchResults ?? []) : []),
  }
  const search = useSearch(core, words, {
    wait: now,
    getTarget: () => target,
    menu: { getEntries: () => [SHOW] },
  })
  await search.setTyped(text)
  await flushPromises()
  return search
}

const listCommands = (target: CommandTarget) => {
  const commands = createCommandPalette({
    core: { searchNames: async () => [], listVaults: async () => ({ vaults: [], showing: '' }) },
    words,
    getTarget: () => target,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => [], applyItemPreview: () => undefined },
    runs: runSupport(),
    getCommands: createAppCommands,
  })
  commands.palette.setOpen(true)
  return commands
}

describe('the actions of the palette', () => {
  const commandIds = new Set([...createAppCommands().map((one) => one.id), SHOW.command])

  const expectCommands = (ids: readonly string[]) => {
    expect(ids.length).toBeGreaterThan(0)
    for (const id of ids) expect(commandIds.has(id), id).toBe(true)
  }

  it('are commands in the list of commands over a note', () => {
    expectCommands(collectActionIds(listCommands(inFront).view.groups.value))
  })

  it('are commands in the list of commands over a window with no note', () => {
    expectCommands(collectActionIds(listCommands(nothingInFront).view.groups.value))
  })

  it('are commands on the rows of a note found by name, by heading and by a passage', async () => {
    const search = await searchFor(
      'coral',
      { names: [byName, heading], searchResults: [noteResult] },
      inFront,
    )

    expectCommands(collectActionIds(search.groups.value))
  })

  it('are commands on the row of a passage of a document', async () => {
    const search = await searchFor('reef', { searchResults: [bookResult] }, inFront)

    const ids = collectActionIds(search.groups.value)
    expect(ids).toStrictEqual([
      'app:file.open#document',
      'app:note.create',
      'app:note.createChild',
      'app:note.createParent',
      'app:note.createJump',
    ])
    expectCommands(ids)
  })

  it('are commands on the rows as the actions that create a note, with a note in front and with none', async () => {
    const created = ['app:note.create']
    const related = ['app:note.createChild', 'app:note.createParent', 'app:note.createJump']
    for (const [target, expected] of [
      [inFront, [...created, ...related]],
      [nothingInFront, created],
    ] as const) {
      const search = await searchFor('Entropy', { names: [byName] }, target)

      const ids = collectActionIds(search.groups.value)
      expect(ids.filter((one) => one.startsWith('app:note.'))).toStrictEqual(expected)
      expectCommands(ids)
    }
  })

  it('are on no row where nothing is found', async () => {
    const search = await searchFor('Entropy', {}, inFront)

    expect(collectActionIds(search.groups.value)).toStrictEqual([])
  })
})
