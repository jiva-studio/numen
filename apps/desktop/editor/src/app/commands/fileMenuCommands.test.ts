/**
 * The menu of the file in front, as the palette offers it, and an entry run as
 * the palette carries it out.
 */
import { describe, expect, it, vi } from 'vitest'
import { commands, runInvocation, ANSWER_WORDS, type CommandDeps } from '@/features/command-palette'
import type { FileMenuEntry } from '@/entities/text-version'
import { WORDS } from '@/shared/words'
import { getNamedIcon } from '@/shared/icons'
import {
  createFileMenuCommands,
  getFileMenuId,
  isOverVersion,
  useFileMenuCommands,
} from './fileMenuCommands'

const recognise: FileMenuEntry = {
  pluginId: 'org.example.ocr',
  pluginName: 'Example OCR',
  command: 'recognise',
  title: 'Read the scan',
  icon: 'scan-text',
  versionId: '',
}
const proofread: FileMenuEntry = {
  pluginId: 'org.example.fix',
  pluginName: 'Example Fix',
  command: 'proofread',
  title: 'Proofread',
  icon: 'spell-check',
  versionId: 'org.example.ocr/recognition',
}
const deleteRecognition: FileMenuEntry = {
  pluginId: 'org.example.ocr',
  pluginName: 'Example OCR',
  command: 'delete-recognition',
  title: 'Delete recognition',
  icon: 'trash-2',
  versionId: 'org.example.ocr/recognition',
}

const front = (file: string, isReady = true) => ({
  tab: 'tab',
  kind: 'document',
  path: '',
  title: '',
  file,
  source: null,
  made: {},
  vault: { id: 'v', name: 'V' },
  isReady,
})

const invoke = (entry: FileMenuEntry, file: string) => ({
  id: getFileMenuId(entry),
  path: '',
  vault: { id: 'v', name: 'V' },
  note: null,
  title: '',
  file,
  others: [],
  name: '',
  kind: null,
  tab: 'tab',
})

const runs = { canRun: () => true, cannotRun: () => {} }

/** A pending request that the test resolves itself. */
const createDeferred = <T>() => {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => (resolve = done))
  return { promise, resolve }
}

describe('file menu commands', () => {
  it('shows the title the plugin declares, with no detail where no other entry has it', () => {
    const [one, two] = createFileMenuCommands('books/Scan.pdf', [recognise, proofread])

    expect(one?.text).toBe('Read the scan')
    expect(one?.detail).toBeUndefined()
    expect(two?.text).toBe('Proofread')
    expect(two?.detail).toBeUndefined()
  })

  it('shows the plugin name as the detail of the entries that share a title', () => {
    const other = { ...deleteRecognition, pluginId: 'org.example.other', pluginName: 'Other OCR' }

    const [first, second, third] = createFileMenuCommands('books/Scan.pdf', [
      deleteRecognition,
      other,
      proofread,
    ])

    expect(first?.detail).toBe('Example OCR')
    expect(second?.detail).toBe('Other OCR')
    expect(third?.detail).toBeUndefined()
  })

  it('files a command over the file as the declared command, and one over a version by the version', () => {
    expect(isOverVersion(recognise)).toBe(false)
    expect(getFileMenuId(recognise)).toBe('plugin-command:org.example.ocr:recognise')
    expect(getFileMenuId(proofread)).toBe(
      'file-menu:org.example.fix:proofread:org.example.ocr/recognition',
    )
  })

  it('is offered only over the loaded file', () => {
    const [one] = createFileMenuCommands('books/Scan.pdf', [recognise])

    expect(one?.group).toBe('file')
    expect(one?.isOffered(front('books/Scan.pdf'), runs)).toBe(true)
    expect(one?.isOffered(front('books/Other.pdf'), runs)).toBe(false)
    expect(one?.isOffered(front('books/Scan.pdf', false), runs)).toBe(false)
    expect(createFileMenuCommands('', [recognise])[0]?.isOffered(front(''), runs)).toBe(false)
  })

  it('draws the icon the plugin names', () => {
    const [made, removed] = createFileMenuCommands('books/Scan.pdf', [recognise, deleteRecognition])

    expect(made?.icon).toBe(getNamedIcon('scan-text'))
    expect(made?.icon).not.toBe(getNamedIcon('no-such-icon'))
    expect(removed?.icon).toBe(getNamedIcon('trash-2'))
  })
})

describe('useFileMenuCommands', () => {
  it('lists only the entries over a version, and runs one and reports it', async () => {
    const get = vi.fn().mockResolvedValue([recognise, proofread])
    const run = vi.fn().mockResolvedValue(undefined)
    const onRun = vi.fn()
    const menu = useFileMenuCommands({ get, run, onRun })
    await menu.load('books/Scan.pdf')
    const writeMessage = vi.fn()

    await runInvocation(
      invoke(proofread, 'books/Scan.pdf'),
      { writeMessage } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(run).toHaveBeenCalledWith('books/Scan.pdf', proofread)
    expect(onRun).toHaveBeenCalledWith('books/Scan.pdf')
    expect(writeMessage).toHaveBeenLastCalledWith(WORDS.commandRan('Proofread'), 'info')
    expect(menu.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getFileMenuId(proofread),
    ])
    expect(commands.get(getFileMenuId(recognise))).toBeUndefined()
  })

  it('reads the menu again after a run', async () => {
    const get = vi.fn().mockResolvedValueOnce([proofread]).mockResolvedValueOnce([])
    const menu = useFileMenuCommands({ get, run: vi.fn().mockResolvedValue(undefined) })
    await menu.load('books/Scan.pdf')

    await commands.get(getFileMenuId(proofread))?.(
      invoke(proofread, 'books/Scan.pdf'),
      { writeMessage: vi.fn() } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(get).toHaveBeenCalledTimes(2)
    expect(menu.paletteCommands.value).toStrictEqual([])
  })

  it('reports a failed run as an error', async () => {
    const menu = useFileMenuCommands({
      get: async () => [proofread],
      run: async () => {
        throw new Error('the plugin did not answer')
      },
    })
    await menu.load('books/Scan.pdf')
    const writeMessage = vi.fn()

    await commands.get(getFileMenuId(proofread))?.(
      invoke(proofread, 'books/Scan.pdf'),
      { writeMessage } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(writeMessage).toHaveBeenCalledWith(expect.stringContaining('not responding'), 'error')
  })

  it('clears the entries for an empty file', async () => {
    const get = vi.fn().mockResolvedValue([proofread])
    const menu = useFileMenuCommands({ get, run: vi.fn() })
    await menu.load('books/Scan.pdf')

    await menu.load('')

    expect(get).toHaveBeenCalledTimes(1)
    expect(menu.paletteCommands.value).toStrictEqual([])
    expect(commands.get(getFileMenuId(proofread))).toBeUndefined()
  })

  it('draws a menu read before at once, and reads it again', async () => {
    const answer = createDeferred<readonly FileMenuEntry[]>()
    const get = vi.fn().mockResolvedValueOnce([proofread]).mockReturnValueOnce(answer.promise)
    const menu = useFileMenuCommands({ get, run: vi.fn() })
    await menu.load('books/Scan.pdf')
    await menu.load('')

    const second = menu.load('books/Scan.pdf')

    expect(menu.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getFileMenuId(proofread),
    ])
    answer.resolve([deleteRecognition])
    await second
    expect(menu.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getFileMenuId(deleteRecognition),
    ])
  })

  it('reads a file once at a time', async () => {
    const answer = createDeferred<readonly FileMenuEntry[]>()
    const get = vi.fn().mockReturnValue(answer.promise)
    const menu = useFileMenuCommands({ get, run: vi.fn() })

    const first = menu.load('books/Scan.pdf')
    const second = menu.load('books/Scan.pdf')
    answer.resolve([proofread])
    await Promise.all([first, second])

    expect(get).toHaveBeenCalledTimes(1)
  })

  it('shows the menu of the latest load when an earlier load finishes after it', async () => {
    const slow = createDeferred<readonly FileMenuEntry[]>()
    const get = vi.fn((file: string) =>
      file === 'books/Slow.pdf' ? slow.promise : Promise.resolve([deleteRecognition]),
    )
    const menu = useFileMenuCommands({ get, run: vi.fn() })

    const first = menu.load('books/Slow.pdf')
    await menu.load('books/Fast.pdf')
    slow.resolve([proofread])
    await first

    expect(menu.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getFileMenuId(deleteRecognition),
    ])
  })

  it('loads and returns commands for a file requested through getForFile', async () => {
    const get = vi.fn().mockResolvedValue([recognise])
    const menu = useFileMenuCommands({ get, run: vi.fn() })

    expect(menu.getForFile('books/Scan.pdf')).toStrictEqual([])
    await Promise.resolve()
    await Promise.resolve()

    expect(get).toHaveBeenCalledWith('books/Scan.pdf')
    expect(menu.getForFile('books/Scan.pdf').map((one) => one.id)).toStrictEqual([
      getFileMenuId(recognise),
    ])
    expect(menu.getIcon(getFileMenuId(recognise))).toBe(getNamedIcon('scan-text'))
  })
})
