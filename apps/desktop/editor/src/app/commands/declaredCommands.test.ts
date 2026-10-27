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

const createSample: DeclaredCommand = {
  pluginId: 'org.example.sample',
  id: 'create-sample',
  title: 'Create a sample',
  contentTypes: [],
  extensions: [],
  arguments: [
    { name: 'title', title: 'Title', isRequired: true },
    { name: 'folder', title: 'Folder', isRequired: false },
  ],
}

const convert: DeclaredCommand = {
  pluginId: 'md.numen.pages',
  id: 'convert',
  title: 'Convert',
  contentTypes: ['application/pdf'],
  extensions: ['pdf'],
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
    const [sample] = createDeclaredPaletteCommands([createSample])

    expect(sample?.id).toBe('plugin-command:org.example.sample:create-sample')
    expect(sample?.group).toBe('window')
    expect(sample?.needs).toBe('arguments')
    expect(sample?.prompts).toStrictEqual([
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
  it('sends the file and values, writes the message and opens the returned path', async () => {
    const run = vi.fn().mockResolvedValue({
      ok: true,
      value: { message: 'Made a sample', openedPath: 'Verbs.md' },
    })
    const declared = useDeclaredCommands({ list: async () => [createSample], run })
    await declared.load()
    const writeMessage = vi.fn()
    const openFile = vi.fn()

    await runInvocation(
      {
        id: getPaletteId(createSample),
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
      { writeMessage, notes: { openFile } } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(run).toHaveBeenCalledWith(createSample, { path: 'a.md', values: { title: 'Verbs' } })
    expect(writeMessage).toHaveBeenLastCalledWith('Made a sample', 'report')
    expect(openFile).toHaveBeenCalledWith('Verbs.md', '', 'here')
    expect(declared.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getPaletteId(createSample),
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
    let commandList: readonly DeclaredCommand[] = [convert]
    const declared = useDeclaredCommands({ list: async () => commandList, run: vi.fn() })
    await declared.load()

    commandList = []
    await declared.load()

    expect(commands.get(getPaletteId(convert))).toBeUndefined()
    expect(declared.paletteCommands.value).toStrictEqual([])
  })
})

describe('a run that opens a file and says nothing', () => {
  it('writes no message', async () => {
    const declared = useDeclaredCommands({
      list: async () => [createSample],
      run: async () => ({ ok: true, value: { message: '', openedPath: 'Verbs.md' } }),
    })
    await declared.load()
    const writeMessage = vi.fn()
    const openFile = vi.fn()

    await commands.get(getPaletteId(createSample))?.(
      {
        id: getPaletteId(createSample),
        path: '',
        vault: { id: 'v', name: 'V' },
        note: null,
        title: '',
        file: '',
        others: [],
        name: '',
        kind: null,
        tab: 'tab',
        values: { title: 'Verbs' },
      },
      { writeMessage, notes: { openFile } } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(writeMessage).not.toHaveBeenCalled()
    expect(openFile).toHaveBeenCalledWith('Verbs.md', '', 'here')
  })
})
