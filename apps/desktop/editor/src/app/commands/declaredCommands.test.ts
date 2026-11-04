/**
 * The commands the plugins declare as the palette offers them, and a run of
 * one as the palette carries it out.
 */
import { describe, expect, it, vi } from 'vitest'
import { commands, runInvocation, ANSWER_WORDS, type CommandDeps } from '@/features/command-palette'
import type { DeclaredCommand } from '@/entities/plugin'
import { getNamedIcon } from '@/shared/icons'
import {
  createDeclaredPaletteCommands,
  getPaletteId,
  useDeclaredCommands,
} from './declaredCommands'

const createSample: DeclaredCommand = {
  pluginId: 'org.example.sample',
  id: 'sample.create',
  title: 'Create a sample',
  icon: 'layers',
  hasBackend: true,
  hasFrontend: false,
  isListedByMenu: false,
  isSuspended: false,
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
  hasBackend: true,
  hasFrontend: false,
  isListedByMenu: false,
  isSuspended: false,
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

    expect(sample?.id).toBe('org.example.sample:sample.create')
    expect(sample?.group).toBe('window')
    expect(sample?.icon).toBe(getNamedIcon('layers'))
    expect(sample?.needs).toBe('arguments')
    expect(sample?.prompts).toStrictEqual([
      { name: 'title', title: 'Title', isRequired: true },
      { name: 'folder', title: 'Folder', isRequired: false },
    ])
  })

  it('over files is offered over a file of its extensions alone', () => {
    const [one] = createDeclaredPaletteCommands([convert])
    const runs = { canRun: () => true, cannotRun: vi.fn() }

    expect(one?.group).toBe('file')
    expect(one?.needs).toBeUndefined()
    expect(one?.isAvailable(front('books/Bees.pdf'), runs)).toBe(true)
    expect(one?.isAvailable(front('books/Bees.epub'), runs)).toBe(false)
    expect(one?.isAvailable(front('books/Bees.pdf', false), runs)).toBe(false)
  })
})

describe('a run of a declared command', () => {
  it('sends the file and values, writes the message and opens the returned path', async () => {
    const run = vi.fn().mockResolvedValue({
      ok: true,
      value: { message: 'Made a sample', openedPath: 'Verbs.md' },
    })
    const declared = useDeclaredCommands({ list: async () => [createSample], run })
    expect(declared.paletteCommands.value).toStrictEqual([])
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
    expect(writeMessage).toHaveBeenLastCalledWith('Made a sample', 'info')
    expect(openFile).toHaveBeenCalledWith('Verbs.md', '', 'current')
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
    expect(openFile).toHaveBeenCalledWith('Verbs.md', '', 'current')
  })

  it('does not open a file when openedPath is empty', async () => {
    const declared = useDeclaredCommands({
      list: async () => [createSample],
      run: async () => ({ ok: true, value: { message: 'Done', openedPath: '' } }),
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
      },
      { writeMessage, notes: { openFile } } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(writeMessage).toHaveBeenCalledWith('Done', 'info')
    expect(openFile).not.toHaveBeenCalled()
  })
})

describe('the declared commands read more than once', () => {
  it('keep the newest list when an earlier read answers last', async () => {
    let answerFirst: (list: readonly DeclaredCommand[]) => void = () => undefined
    const first = new Promise<readonly DeclaredCommand[]>((resolve) => {
      answerFirst = resolve
    })
    const lists = [first, Promise.resolve([convert])]
    const declared = useDeclaredCommands({
      list: () => lists.shift() as Promise<readonly DeclaredCommand[]>,
    })

    const slow = declared.load()
    await declared.load()
    answerFirst([createSample])
    await slow

    expect(declared.declarations.value?.map((one) => one.id)).toStrictEqual(['convert'])
  })

  it('are none until the first read has answered', async () => {
    const declared = useDeclaredCommands({ list: async () => [convert] })

    expect(declared.declarations.value).toBeUndefined()
    await declared.load()

    expect(declared.declarations.value).toHaveLength(1)
  })
})

describe('a declared command the palette leaves out', () => {
  it('is one of a suspended plugin, and one a menu lists', async () => {
    const declared = useDeclaredCommands({
      list: async () => [
        { ...createSample, isSuspended: true },
        { ...convert, isListedByMenu: true },
      ],
    })
    await declared.load()

    expect(declared.paletteCommands.value).toStrictEqual([])
    expect(declared.declarations.value).toHaveLength(2)
  })
})
