/**
 * The commands the plugins declare as the palette offers them, and a run of
 * one as the palette carries it out.
 */
import { describe, expect, it, vi } from 'vitest'
import { createWhenReader } from '@numen/plugin-loader'
import { commands, runInvocation, ANSWER_WORDS, type CommandDeps } from '@/features/command-palette'
import type { DeclaredCommand, DeclaredList } from '@/entities/plugin'
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
  arguments: [
    { name: 'title', title: 'Title', isRequired: true },
    { name: 'folder', title: 'Folder', isRequired: false },
  ],
}

const convert: DeclaredCommand = {
  pluginId: 'md.numen.pages',
  id: 'convert',
  title: 'Convert',
  when: "target.contentType == 'application/pdf'",
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
  it('asks for its required arguments, one step each, and leaves the optional ones out', () => {
    const [sample] = createDeclaredPaletteCommands([createSample])

    expect(sample?.id).toBe('org.example.sample:sample.create')
    expect(sample?.group).toBe('window')
    expect(sample?.icon).toBe(getNamedIcon('layers'))
    expect(sample?.input).toBe('arguments')
    expect(sample?.prompts).toStrictEqual([{ name: 'title', title: 'Title', isRequired: true }])
  })

  it('asks nothing when every itemId is optional', () => {
    const [one] = createDeclaredPaletteCommands([
      { ...createSample, arguments: [{ name: 'folder', title: 'Folder', isRequired: false }] },
    ])

    expect(one?.input).toBeUndefined()
    expect(one?.prompts).toBeUndefined()
  })

  it('over files is offered over a file of the content type its condition names alone', () => {
    const [one] = createDeclaredPaletteCommands([convert], {
      getContentTypes: () => ({ pdf: 'application/pdf', epub: 'application/epub+zip' }),
      getConfig: () => ({}),
      whenReader: createWhenReader(),
    })
    const runs = { canRun: () => true, cannotRun: vi.fn() }

    expect(one?.group).toBe('file')
    expect(one?.input).toBeUndefined()
    expect(one?.isAvailable(front('books/Bees.pdf'), runs)).toBe(true)
    expect(one?.isAvailable(front('books/Bees.epub'), runs)).toBe(false)
    expect(one?.isAvailable(front('books/Bees.pdf', false), runs)).toBe(false)
  })
})

describe('a declared command with items', () => {
  const proofread: DeclaredCommand = {
    pluginId: 'org.example.ink',
    id: 'ink.proofread',
    title: 'Proofread text',
    when: 'version.isLive',
    hasItems: true,
    arguments: [],
  }
  const invocation = (name: string) => ({
    id: getPaletteId(proofread),
    path: 'talks/a.mp3',
    vault: { id: 'v', name: 'V' },
    note: null,
    title: '',
    file: 'talks/a.mp3',
    others: [],
    name,
    kind: null,
    tab: 'tab',
  })

  it('asks for one of its items, which it lists over the file of the target', async () => {
    const listItems = vi.fn(async () => [{ id: 'org.example.ears/ears', title: 'transcript' }])
    const [one] = createDeclaredPaletteCommands([proofread], undefined, listItems)

    expect(one?.input).toBe('items')
    expect(one?.isAvailable(front('talks/a.mp3'), { canRun: () => true, cannotRun: vi.fn() })).toBe(
      true,
    )
    expect(one?.isAvailable(front(''), { canRun: () => true, cannotRun: vi.fn() })).toBe(false)
    expect(await one?.listItems?.(front('talks/a.mp3'))).toStrictEqual([
      { id: 'org.example.ears/ears', title: 'transcript' },
    ])
    expect(listItems).toHaveBeenCalledWith(proofread, 'talks/a.mp3')
  })

  it('runs over the file with the item picked as its itemId', async () => {
    const run = vi.fn().mockResolvedValue({ ok: true, value: { message: '', openedPath: '' } })
    const declared = useDeclaredCommands({
      list: async () => ({ commands: [proofread], slots: [] }),
      run,
    })
    await declared.load()

    await runInvocation(
      invocation('org.example.ears/ears'),
      { writeMessage: vi.fn(), notes: { openFile: vi.fn() } } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(run).toHaveBeenCalledWith(
      proofread,
      {
        path: 'talks/a.mp3',
        values: {},
        itemId: 'org.example.ears/ears',
      },
      expect.anything(),
    )
  })
})

describe('a run of a declared command', () => {
  it('sends that a place named its target', async () => {
    const run = vi.fn().mockResolvedValue({ ok: true, value: { message: '', openedPath: '' } })
    const declared = useDeclaredCommands({
      list: async () => ({ commands: [createSample], slots: [] }),
      run,
    })
    await declared.load()

    await runInvocation(
      {
        id: getPaletteId(createSample),
        path: 'Glaciers',
        vault: { id: 'v', name: 'V' },
        note: null,
        title: 'Glaciers',
        file: 'Glaciers',
        others: [],
        name: '',
        kind: null,
        tab: '',
        values: { title: 'Moraines' },
        isTargetNamed: true,
      },
      { writeMessage: vi.fn(), notes: { openFile: vi.fn() } } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(run).toHaveBeenCalledWith(
      createSample,
      {
        path: 'Glaciers',
        values: { title: 'Moraines' },
        isTargetNamed: true,
      },
      expect.anything(),
    )
  })

  it('sends the file and values, writes the message and opens the returned path', async () => {
    const run = vi.fn().mockResolvedValue({
      ok: true,
      value: { message: 'Made a sample', openedPath: 'Verbs.md' },
    })
    const declared = useDeclaredCommands({
      list: async () => ({ commands: [createSample], slots: [] }),
      run,
    })
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

    expect(run).toHaveBeenCalledWith(
      createSample,
      { path: 'a.md', values: { title: 'Verbs' } },
      expect.anything(),
    )
    expect(writeMessage).toHaveBeenLastCalledWith('Made a sample', 'info')
    expect(openFile).toHaveBeenCalledWith('Verbs.md', '', 'current')
    expect(declared.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getPaletteId(createSample),
    ])
  })

  it('that failed says why as an error', async () => {
    const declared = useDeclaredCommands({
      list: async () => ({ commands: [convert], slots: [] }),
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
    const declared = useDeclaredCommands({
      list: async () => ({ commands: commandList, slots: [] }),
      run: vi.fn(),
    })
    await declared.load()

    commandList = []
    await declared.load()

    expect(commands.get(getPaletteId(convert))).toBeUndefined()
    expect(declared.paletteCommands.value).toStrictEqual([])
  })
})

describe('a run that opens a file', () => {
  it('answers the path it opened', async () => {
    const declared = useDeclaredCommands({
      list: async () => ({ commands: [createSample], slots: [] }),
      run: async () => ({ ok: true, value: { message: '', openedPath: 'Verbs.md' } }),
    })
    await declared.load()

    const path = await commands.get(getPaletteId(createSample))?.(
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
      { writeMessage: vi.fn(), notes: { openFile: vi.fn() } } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(path).toBe('Verbs.md')
  })
})

describe('a run that opens a file and says nothing', () => {
  it('writes no message', async () => {
    const declared = useDeclaredCommands({
      list: async () => ({ commands: [createSample], slots: [] }),
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
      list: async () => ({ commands: [createSample], slots: [] }),
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
    let answerFirst: (list: DeclaredList) => void = () => undefined
    const first = new Promise<DeclaredList>((resolve) => {
      answerFirst = resolve
    })
    const lists = [first, Promise.resolve({ commands: [convert], slots: [] })]
    const declared = useDeclaredCommands({
      list: () => lists.shift() as Promise<DeclaredList>,
    })

    const slow = declared.load()
    await declared.load()
    answerFirst({ commands: [createSample], slots: [] })
    await slow

    expect(declared.declarations.value?.map((one) => one.id)).toStrictEqual(['convert'])
  })

  it('keep the lists they hold, and register no command again, when a read answers the same', async () => {
    const declared = useDeclaredCommands({
      list: async () => ({
        commands: [{ ...createSample, arguments: [...createSample.arguments] }],
        slots: [
          {
            pluginId: 'org.example.sample',
            slot: 'titleBar.commands',
            command: 'sample.create',
          },
        ],
      }),
    })
    await declared.load()
    const commandsBefore = declared.commands.value
    const slotsBefore = declared.slots.value
    const paletteBefore = declared.paletteCommands.value

    await declared.load()

    expect(declared.commands.value).toBe(commandsBefore)
    expect(declared.slots.value).toStrictEqual(slotsBefore)
    expect(declared.paletteCommands.value).toBe(paletteBefore)
  })

  it('take the new list when a read answers another one', async () => {
    const lists = [
      { commands: [createSample], slots: [] },
      { commands: [convert], slots: [] },
    ]
    const declared = useDeclaredCommands({ list: async () => lists.shift() as DeclaredList })
    await declared.load()

    await declared.load()

    expect(declared.commands.value?.map((one) => one.id)).toStrictEqual(['convert'])
  })

  it('resolves waitForFirstList after the first list arrives', async () => {
    let answer: (list: DeclaredList) => void = () => undefined
    const declared = useDeclaredCommands({
      list: () => new Promise<DeclaredList>((resolve) => (answer = resolve)),
    })
    let isSettled = false
    const firstList = declared.waitForFirstList().then(() => (isSettled = true))

    const loading = declared.load()
    await Promise.resolve()
    expect(isSettled).toBe(false)
    answer({ commands: [convert], slots: [] })
    await Promise.all([loading, firstList])

    expect(isSettled).toBe(true)
    expect(declared.commands.value).toHaveLength(1)
  })

  it('settle the wait for the first list when the read fails', async () => {
    const declared = useDeclaredCommands({ list: () => Promise.reject(new Error('core gone')) })

    await expect(declared.load()).rejects.toThrow('core gone')

    await expect(declared.waitForFirstList()).resolves.toBeUndefined()
  })

  it('are none until the first read has answered', async () => {
    const declared = useDeclaredCommands({ list: async () => ({ commands: [convert], slots: [] }) })

    expect(declared.declarations.value).toBeUndefined()
    await declared.load()

    expect(declared.declarations.value).toHaveLength(1)
  })
})

describe('a declared command the palette leaves out', () => {
  it('is one the plugin host lists', async () => {
    const declared = useDeclaredCommands({
      list: async () => ({ commands: [createSample, convert], slots: [] }),
      isListed: () => true,
    })
    await declared.load()

    expect(declared.paletteCommands.value).toStrictEqual([])
    expect(declared.declarations.value).toHaveLength(2)
  })
})
