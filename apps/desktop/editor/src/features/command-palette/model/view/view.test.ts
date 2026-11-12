/**
 * What the palette renders over the commands, without a window: which rows
 * remain for the typed words, which are highlighted, and what a group shows while it
 * is empty.
 */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { File } from '@lucide/vue'
import { getNoteIcon } from '@/entities/note'
import { type Command } from '@/entities/command'
import { findIcon, COMMAND, type CommandTarget } from '@/shared/model/command'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../../words'
import { createAppCommands } from '@/testing/appCommands'
import type { PendingStep } from '../../lib/step/step'
import type { PaletteLists } from '../../types'
import type { Commands } from '../palette/palette'
import { runSupport } from '../palette/runSupport'
import { findRowIcon } from '../rowIcons'
import type { SearchState } from '../search/search'
import { createStepGroups } from './stepGroups'
import type { ViewState } from './viewState'
import { getPaletteView } from './view'

const words = { ...windowWords, ...PALETTE_WORDS }

const front = (overrides: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...overrides,
})

const lists: PaletteLists = {
  getStepGroups: () => [],
  previewItem: vi.fn(),
}

const createView = (commands: readonly Command[]) =>
  getPaletteView({
    words,
    commands,
    byId: new Map(commands.map((one) => [one.id, one])),
    runs: runSupport(),
    lists,
    found: ref([]),
    known: ref([]),
    showing: ref(''),
    isWorking: ref(false),
    failureMessage: ref(''),
    getStepTitle: () => 'Ontology',
    getStepLabel: () => 'Ontology.md',
  })

const createCommand = (overrides: Partial<Command> & Pick<Command, 'id' | 'text'>): Command => ({
  group: 'note',
  isAvailable: () => true,
  ...overrides,
})

const getNoteGroup = (view: ReturnType<typeof createView>, target: CommandTarget, typed: string) =>
  view.createGroups(null, target, typed).find((group) => group.id === 'note')

describe('the commands as the words typed leave them', () => {
  const view = createView(createAppCommands())

  it('lights nothing where nothing is typed', () => {
    const readItem = getNoteGroup(view, front(), '')?.items.find(
      (item) => item.id === 'app:file.open',
    )

    expect(readItem).toBeDefined()
    expect(readItem).not.toHaveProperty('at')
  })

  it('lights the words typed where the text of a command begins with them', () => {
    const word = words.read.toLowerCase().slice(0, 3)

    const readItem = getNoteGroup(view, front(), word)?.items.find(
      (item) => item.id === 'app:file.open',
    )

    expect(readItem?.at).toStrictEqual([{ start: 0, end: 3 }])
  })

  it('lights the words typed where they match inside the text of a command', () => {
    const readItem = getNoteGroup(view, front(), words.read.toLowerCase().slice(1, 3))?.items.find(
      (item) => item.id === 'app:file.open',
    )

    expect(readItem?.at).toStrictEqual([{ start: 1, end: 3 }])
  })

  it('leaves out every command the words typed are not in', () => {
    expect(getNoteGroup(view, front(), 'zzzqqq')).toBeUndefined()
  })

  it('reads the words typed around spaces and in any case', () => {
    const word = words.read.toLowerCase().slice(0, 3).toUpperCase()

    const ids = getNoteGroup(view, front(), `   ${word}  `)?.items.map((item) => item.id)

    expect(ids).toContain('app:file.open')
  })
})

describe('the command reached on the same row', () => {
  it('is drawn as the second action where it is offered', () => {
    const view = createView([
      createCommand({ id: 'first', text: 'First', also: 'second' }),
      createCommand({ id: 'second', text: 'Second' }),
    ])

    expect(getNoteGroup(view, front(), '')?.items[0]?.actions).toStrictEqual([
      { id: 'first', text: 'First' },
      { id: 'second', text: 'Second' },
    ])
  })

  it('is drawn nowhere where it is not offered', () => {
    const view = createView([
      createCommand({ id: 'first', text: 'First', also: 'second' }),
      createCommand({ id: 'second', text: 'Second', isAvailable: () => false }),
    ])

    expect(getNoteGroup(view, front(), '')?.items[0]?.actions).toStrictEqual([
      { id: 'first', text: 'First' },
    ])
  })
})

describe('empty command groups', () => {
  const view = createView(createAppCommands())

  it('omits groups that hold no commands', () => {
    expect(getNoteGroup(view, front({ isReady: false }), '')).toBeUndefined()
    expect(getNoteGroup(view, front({ path: '' }), '')).toBeUndefined()
  })
})

describe('a command that gives no words for its confirmation', () => {
  const command = createCommand({ id: 'bare', text: 'Bare' })
  const step = (which: PendingStep['step']): PendingStep => ({
    step: which,
    command,
    on: front(),
  })
  const view = createView([command])

  it('draws the two answers with empty words', () => {
    const items = view.createGroups(step('confirmation'), front(), '')[0]?.items

    expect(items).toStrictEqual([
      {
        id: expect.any(String),
        title: '',
        detail: '',
        actions: [{ id: expect.any(String), text: '' }],
      },
      {
        id: expect.any(String),
        title: ' Ontology.md',
        detail: '',
        actions: [{ id: expect.any(String), text: '' }],
      },
    ])
  })

  it('draws the name to type back with empty words', () => {
    const items = view.createGroups(step('retypedName'), front(), 'Ontology')[0]?.items

    expect(items).toStrictEqual([
      {
        id: expect.any(String),
        title: ' “Ontology”',
        detail: '',
        isDisabled: false,
        actions: [{ id: expect.any(String), text: '' }],
      },
    ])
  })

  it('draws asking group with id asking and arguments group with default title when prompts are absent', () => {
    const askingGroup = view.createGroups(step('confirmation'), front(), '')[0]
    expect(askingGroup?.id).toBe('asking')

    const argGroupNoPrompt = view.createGroups(step('arguments'), front(), 'val')[0]
    expect(argGroupNoPrompt?.id).toBe('arguments')
    expect(argGroupNoPrompt?.title).toBe('')
    expect(argGroupNoPrompt?.items).toStrictEqual([
      {
        id: expect.any(String),
        title: ': “val”',
        actions: [{ id: expect.any(String), text: words.givesIt }],
      },
    ])

    const cmdWithOptPrompt = createCommand({
      id: 'opt',
      text: 'Opt',
      prompts: [{ name: 'topic', title: 'Topic', isRequired: false }],
    })
    const optView = createView([cmdWithOptPrompt])
    const optGroup = optView.createGroups(
      { step: 'arguments', command: cmdWithOptPrompt, on: front() },
      front(),
      '',
    )[0]
    expect(optGroup?.items).toStrictEqual([
      {
        id: expect.any(String),
        title: `${words.leavesOut} Topic`,
        actions: [{ id: expect.any(String), text: words.leavesOut }],
      },
    ])
  })

  it('matches commands by detail without highlight ranges and renders file and settings groups', () => {
    const customWords = { ...words, overSettings: 'Over Settings', settings: 'Fallback Settings' }
    const cmds: readonly Command[] = [
      createCommand({ id: 'det', text: 'Alpha', detail: 'Special detail' }),
      createCommand({ id: 'f1', text: 'File cmd', group: 'file' }),
      createCommand({ id: 's1', text: 'Set cmd', group: 'settings' }),
    ]
    const customView = getPaletteView({
      words: customWords,
      commands: cmds,
      byId: new Map(cmds.map((one) => [one.id, one])),
      runs: runSupport(),
      lists,
      found: ref([]),
      known: ref([]),
      showing: ref(''),
      isWorking: ref(false),
      failureMessage: ref(''),
      getStepTitle: () => 'Ontology',
      getStepLabel: () => 'Ontology.md',
    })

    const matchedByDetail = customView.createGroups(null, front(), 'special')
    const detailItem = matchedByDetail.find((g) => g.id === 'note')?.items[0]
    expect(detailItem?.id).toBe('det')
    expect(detailItem?.detail).toBe('Special detail')
    expect(detailItem).not.toHaveProperty('at')

    const allGroups = customView.createGroups(null, front(), '')
    expect(allGroups.find((g) => g.id === 'file')?.title).toBe(customWords.overFile)
    expect(allGroups.find((g) => g.id === 'settings')?.title).toBe('Over Settings')

    const fallbackWords = {
      ...words,
      overSettings: undefined as unknown as string,
      settings: 'Fallback Settings',
    }
    const fallbackView = getPaletteView({
      words: fallbackWords,
      commands: cmds,
      byId: new Map(cmds.map((one) => [one.id, one])),
      runs: runSupport(),
      lists,
      found: ref([]),
      known: ref([]),
      showing: ref(''),
      isWorking: ref(false),
      failureMessage: ref(''),
      getStepTitle: () => 'Ontology',
      getStepLabel: () => 'Ontology.md',
    })
    expect(
      fallbackView.createGroups(null, front(), '').find((g) => g.id === 'settings')?.title,
    ).toBe('Fallback Settings')
  })
})

describe('findRowIcon', () => {
  const customIcon = defineComponent({ name: 'CustomIcon', render: () => null })

  it('resolves command and note icons when commands palette is open', () => {
    const commands = {
      palette: { open: ref(true) },
      view: {
        findTabKind: (id: string) => (id === 'n1' ? 'sample' : null),
        findCommandIcon: (id: string) => (id === 'custom-cmd' ? customIcon : undefined),
      },
    } as unknown as Commands
    const search = {
      reads: { findTabKind: () => 'form', findHitFile: () => 'doc.pdf' },
    } as unknown as SearchState

    expect(findRowIcon('n1', commands, search)).toBe(getNoteIcon('sample'))
    expect(findRowIcon('custom-cmd', commands, search)).toBe(customIcon)
    expect(findRowIcon('app:note.create', commands, search)).toBe(findIcon('app:note.create'))
  })

  it('resolves search note type, and file hit icons when commands palette is closed', () => {
    const commands = {
      palette: { open: ref(false) },
      view: { findTabKind: () => 'sample', findCommandIcon: () => customIcon },
    } as unknown as Commands
    const search = {
      reads: {
        findTabKind: (id: string) => (id === 'hit-note' ? 'sheet' : null),
        findHitFile: (id: string) => (id === 'hit-file' ? 'book.epub' : null),
      },
    } as unknown as SearchState

    expect(findRowIcon('hit-note', commands, search)).toBe(getNoteIcon('sheet'))
    expect(findRowIcon('hit-file', commands, search, () => customIcon)).toBe(customIcon)
    expect(findRowIcon('hit-file', commands, search, () => undefined)).toBe(File)
    expect(findRowIcon('hit-file', commands, search)).toBe(File)
    expect(findRowIcon('unknown', commands, search)).toBeNull()
  })
})

describe('the icon of the command that opens a document', () => {
  it('is the file icon the row of a document is drawn with', () => {
    expect(findIcon('app:file.open#document')).toBe(File)
  })

  it('is the icon of the command the row of a document offers', () => {
    const commands = {
      palette: { open: ref(true) },
      view: { findTabKind: () => null, findCommandIcon: () => null },
    }
    const search = { reads: { findTabKind: () => null, findHitFile: () => null } }

    expect(
      findRowIcon(
        'app:file.open#document',
        commands as unknown as Commands,
        search as unknown as SearchState,
      ),
    ).toBe(File)
  })
})

describe('the two choices of the confirmation that empties the trash', () => {
  const emptyTrash = createAppCommands().find((one) => one.id === COMMAND.emptyTrash)

  it('are both drawn with an icon', () => {
    const groups = createStepGroups({
      words,
      getStepLabel: () => 'trash',
    } as unknown as ViewState)
    const step: PendingStep = {
      step: 'confirmation',
      command: emptyTrash as Command,
      on: front(),
    }
    const commands = {
      palette: { open: ref(true) },
      view: { findTabKind: () => null, findCommandIcon: () => null },
    }
    const search = { reads: { findTabKind: () => null, findHitFile: () => null } }

    const items = groups.getConfirmGroup(step, '').items

    expect(items.map((one) => one.id)).toStrictEqual(['app:trash.empty:keep', COMMAND.emptyTrash])
    for (const one of items) {
      expect(
        findRowIcon(one.id, commands as unknown as Commands, search as unknown as SearchState),
      ).not.toBeNull()
    }
  })
})
