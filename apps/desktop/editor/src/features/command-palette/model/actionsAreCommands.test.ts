/**
 * Every action the palette draws on a row is the id of a command, and the source holds no branch
 * that opens a file or creates a note for a chosen result on a path of its own.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve, sep } from 'node:path'
import { describe, expect, it } from 'vitest'
import type { PaletteGroup } from '@numen/workbench'
import type { SlotCommand } from '@numen/plugin-loader'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../words'
import { createAppCommands } from '@/testing/appCommands'
import { runSupport } from './palette/runSupport'
import { createCommandPalette } from './palette/palette'
import { useSearch, type NameMatch, type Passage, type SearchDeps } from './search/search'
import type { CommandTarget } from '@/entities/command'

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

const collectActionIds = (groups: readonly PaletteGroup[]): string[] =>
  groups.flatMap((group) =>
    group.items.flatMap((one) => (one.actions ?? [{ id: one.id }]).map((action) => action.id)),
  )

const searchFor = async (
  text: string,
  found: { names?: NameMatch[]; passages?: Passage[] },
  target: CommandTarget,
) => {
  const core: SearchDeps = {
    names: async () => found.names ?? [],
    search: async (_query, mode) => (mode === 'words' ? (found.passages ?? []) : []),
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
    core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
    words,
    at: () => target,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => [], previewItem: () => undefined },
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
      { names: [byName, heading], passages: [notePassage] },
      inFront,
    )

    expectCommands(collectActionIds(search.groups.value))
  })

  it('are commands on the row of a passage of a document', async () => {
    const search = await searchFor('reef', { passages: [bookPassage] }, inFront)

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

const readSources = (): Record<string, string> => {
  const root = resolve(import.meta.dirname, '../../..')
  const found: Record<string, string> = {}
  for (const entry of readdirSync(root, { recursive: true, withFileTypes: true })) {
    if (!entry.isFile() || !/\.(ts|vue)$/.test(entry.name)) continue
    const file = join(entry.parentPath, entry.name)
    found[`/src/${relative(root, file).split(sep).join('/')}`] = readFileSync(file, 'utf8')
  }
  return found
}

const sources = readSources()

describe('the source of the editor', () => {
  const thisFile = '/src/features/command-palette/model/actionsAreCommands.test.ts'
  const files = Object.entries(sources).filter(([path]) => path !== thisFile)

  const findIn = (pattern: RegExp) =>
    files.filter(([, text]) => pattern.test(text)).map(([path]) => path)

  it('holds files to look at', () => {
    expect(files.length).toBeGreaterThan(100)
  })

  it('has no destination that a chosen result opens on a path of its own', () => {
    const names = ['open' + 'Destination', 'resolve' + 'Destination', 'Search' + 'Destination']

    expect(findIn(new RegExp(`\\b(${names.join('|')})\\b`))).toStrictEqual([])
  })

  it('has no function that creates a note for a chosen result outside the commands', () => {
    expect(
      findIn(new RegExp(`\\b(create${'NoteInvocation'}|CREATE_NOTE${'_ID'})\\b`)),
    ).toStrictEqual([])
  })

  it('has no action id for a document that is no command, in the palette', () => {
    const palette = files.filter(([path]) => path.startsWith('/src/features/command-palette/'))

    expect(
      palette.filter(([, text]) => new RegExp(`\\bNOTE = ${"'note'"}`).test(text)),
    ).toStrictEqual([])
  })

  it('has no prefix that makes the id of an action differ from the id of its command', () => {
    expect(
      findIn(new RegExp(`\\b(MENU_ACTION${'_PREFIX'}|resolveMenu${'Choice'})\\b`)),
    ).toStrictEqual([])
  })

  it('runs a chosen result in the field only through the choice it is handed', () => {
    const view = sources['/src/features/command-palette/ui/CommandPalette.vue'] ?? ''
    const field = sources['/src/features/command-palette/model/usePaletteInput.ts'] ?? ''

    expect(field).not.toBe('')
    for (const text of [view, field]) {
      expect(text).not.toMatch(/CREATE_NOTE/)
      expect(text).not.toMatch(/item\s*===/)
    }
    expect(view).not.toMatch(/runInvocation\(/)
    expect(field.match(/runInvocation\(/g) ?? []).toHaveLength(1)
  })
})
