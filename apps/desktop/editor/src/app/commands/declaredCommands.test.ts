/**
 * The commands the plugins declare as the palette offers them, and a run of
 * one as the palette carries it out.
 */
import { describe, expect, it, vi } from 'vitest'
import { commands, runInvocation, ANSWER_WORDS, type CommandDeps } from '@/features/command-palette'
import type { DeclaredCommand } from '@/entities/plugin'
import {
  createDeclaredPaletteCommands,
  getPaletteId,
  useDeclaredCommands,
} from './declaredCommands'

const createDeck: DeclaredCommand = {
  pluginId: 'md.numen.cards',
  id: 'create-deck',
  title: 'Create a deck',
  description: '',
  fileExtensions: [],
  arguments: [
    { name: 'title', title: 'Title', type: 'string', isRequired: true },
    { name: 'folder', title: 'Folder', type: 'string', isRequired: false },
  ],
}

const convert: DeclaredCommand = {
  pluginId: 'md.numen.pages',
  id: 'convert',
  title: 'Convert',
  description: '',
  fileExtensions: ['pdf'],
  arguments: [],
}

const front = (file: string, isReady = true) => ({
  tab: 'tab',
  kind: 'note',
  path: file,
  title: '',
  file,
  source: null,
  made: {},
  vault: { id: 'v', name: 'V' },
  isReady,
})

describe('a declared command in the palette', () => {
  it('asks for its arguments, one step each', () => {
    const [deck] = createDeclaredPaletteCommands([createDeck])

    expect(deck?.id).toBe('plugin-command:md.numen.cards:create-deck')
    expect(deck?.group).toBe('window')
    expect(deck?.needs).toBe('arguments')
    expect(deck?.prompts).toStrictEqual([
      { name: 'title', title: 'Title', isRequired: true },
      { name: 'folder', title: 'Folder', isRequired: false },
    ])
  })

  it('over files is offered over a file of its extensions alone', () => {
    const [one] = createDeclaredPaletteCommands([convert])
    const runs = { canRun: () => true, cannotRun: () => {} }

    expect(one?.group).toBe('file')
    expect(one?.needs).toBeUndefined()
    expect(one?.isOffered(front('books/Bees.pdf'), runs)).toBe(true)
    expect(one?.isOffered(front('books/Bees.epub'), runs)).toBe(false)
    expect(one?.isOffered(front('books/Bees.pdf', false), runs)).toBe(false)
  })
})

describe('a run of a declared command', () => {
  it('hands the plugin the file and the values, and says what it answered', async () => {
    const run = vi.fn().mockResolvedValue({ ok: true, value: 'Created the deck Verbs.md' })
    const declared = useDeclaredCommands({ list: async () => [createDeck], run })
    await declared.load()
    const writeMessage = vi.fn()

    await runInvocation(
      {
        id: getPaletteId(createDeck),
        path: '',
        vault: { id: 'v', name: 'V' },
        note: null,
        title: '',
        file: 'a.md',
        others: [],
        name: '',
        kind: null,
        tab: 'tab',
        values: { title: 'Verbs' },
      },
      { writeMessage } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(run).toHaveBeenCalledWith(createDeck, { path: 'a.md', values: { title: 'Verbs' } })
    expect(writeMessage).toHaveBeenLastCalledWith('Created the deck Verbs.md', 'report')
    expect(declared.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getPaletteId(createDeck),
    ])
  })

  it('that failed says why as an error', async () => {
    const declared = useDeclaredCommands({
      list: async () => [convert],
      run: async () => ({ ok: false, error: 'the disk is full' }),
    })
    await declared.load()
    const writeMessage = vi.fn()

    await commands.get(getPaletteId(convert))?.(
      {
        id: getPaletteId(convert),
        path: 'a.pdf',
        vault: { id: 'v', name: 'V' },
        note: null,
        title: '',
        file: 'a.pdf',
        others: [],
        name: '',
        kind: null,
        tab: 'tab',
      },
      { writeMessage } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(writeMessage).toHaveBeenLastCalledWith('the disk is full', 'error')
  })

  it('is no longer handled once the plugins no longer declare it', async () => {
    let listed: readonly DeclaredCommand[] = [convert]
    const declared = useDeclaredCommands({ list: async () => listed, run: vi.fn() })
    await declared.load()

    listed = []
    await declared.load()

    expect(commands.get(getPaletteId(convert))).toBeUndefined()
    expect(declared.paletteCommands.value).toStrictEqual([])
  })
})
