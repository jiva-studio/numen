/**
 * What the palette draws over the commands, without a window: which rows the
 * words typed leave in, where they are lit, and what a group says while it
 * holds nothing.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { WORDS as words } from '@/shared/words'
import { createCommands } from '../lib/commands'
import type { PendingStep } from '../lib/step'
import type { Command, CommandTarget, PaletteLists } from '../types'
import { runSupport } from './runSupport'
import { getPaletteView } from './view'

const front = (overrides: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  source: null,
  made: {},
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...overrides,
})

const lists: PaletteLists = {
  getStepGroups: () => [],
  previewItem: () => {},
}

/** The view over a set of commands, with every list empty. */
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
  isOffered: () => true,
  ...overrides,
})

const getNoteGroup = (view: ReturnType<typeof createView>, target: CommandTarget, typed: string) =>
  view.createGroups(null, target, typed).find((group) => group.id === 'note')

describe('the commands as the words typed leave them', () => {
  const view = createView(createCommands(words))

  it('lights nothing where nothing is typed', () => {
    const readItem = getNoteGroup(view, front(), '')?.items.find((item) => item.id === 'read')

    expect(readItem).toBeDefined()
    expect(readItem).not.toHaveProperty('at')
  })

  it('lights the words typed where the text of a command begins with them', () => {
    const word = words.read.toLowerCase().slice(0, 3)

    const readItem = getNoteGroup(view, front(), word)?.items.find((item) => item.id === 'read')

    expect(readItem?.at).toStrictEqual([{ from: 0, to: 3 }])
  })

  it('lights the words typed where they stand inside the text of a command', () => {
    const readItem = getNoteGroup(view, front(), words.read.toLowerCase().slice(1, 3))?.items.find(
      (item) => item.id === 'read',
    )

    expect(readItem?.at).toStrictEqual([{ from: 1, to: 3 }])
  })

  it('leaves out every command the words typed are not in', () => {
    expect(getNoteGroup(view, front(), 'zzzqqq')?.items).toStrictEqual([])
  })

  it('reads the words typed around spaces and in any case', () => {
    const word = words.read.toLowerCase().slice(0, 3).toUpperCase()

    const ids = getNoteGroup(view, front(), `   ${word}  `)?.items.map((item) => item.id)

    expect(ids).toContain('read')
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
      createCommand({ id: 'second', text: 'Second', isOffered: () => false }),
    ])

    expect(getNoteGroup(view, front(), '')?.items[0]?.actions).toStrictEqual([
      { id: 'first', text: 'First' },
    ])
  })
})

describe('what the group over a note says while it holds nothing', () => {
  const view = createView(createCommands(words))

  it('says the vault could not be opened where it is not ready', () => {
    expect(getNoteGroup(view, front({ isReady: false }), '')?.silence).toBe(words.noVault)
  })

  it('says nothing was found where a note is in front', () => {
    expect(getNoteGroup(view, front(), '')?.silence).toBe(words.noneFound)
  })

  it('says no note is in front where none is', () => {
    expect(getNoteGroup(view, front({ path: '' }), '')?.silence).toBe(words.noNote)
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
    const items = view.createGroups(step('asking'), front(), '')[0]?.items

    expect(items).toStrictEqual([
      { id: 'no', title: '', detail: '', actions: [{ id: 'no', text: '' }] },
      {
        id: 'yes',
        title: ' Ontology.md',
        detail: '',
        actions: [{ id: 'yes', text: '' }],
      },
    ])
  })

  it('draws the name to type back with empty words', () => {
    const items = view.createGroups(step('exactly'), front(), 'Ontology')[0]?.items

    expect(items).toStrictEqual([
      {
        id: 'exactly',
        title: ' “Ontology”',
        detail: '',
        isDisabled: false,
        actions: [{ id: 'exactly', text: '' }],
      },
    ])
  })
})
