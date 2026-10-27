/**
 * The menu the text versions of the file in front make, as the palette offers
 * it, and an entry run as the palette carries it out.
 */
import { describe, expect, it, vi } from 'vitest'
import { commands, runInvocation, ANSWER_WORDS, type CommandDeps } from '@/features/command-palette'
import type { FileMenuEntry } from '@/entities/text-version'
import { WORDS } from '@/shared/words'
import { findIcon, getNamedIcon } from '@/shared/icons'
import {
  createFileMenuPaletteCommands,
  getFileMenuId,
  getFileMenuText,
  useFileMenuCommands,
} from './fileMenuCommands'

const recognise: FileMenuEntry = {
  action: 'create-text',
  pluginId: 'org.example.ocr',
  pluginName: 'Example OCR',
  title: 'Read the scan',
  icon: 'scan-text',
  versionId: '',
}
const deleteVersion: FileMenuEntry = {
  action: 'delete-version',
  pluginId: 'org.example.ocr',
  pluginName: 'Example OCR',
  title: '',
  icon: '',
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

describe('file menu palette commands', () => {
  it('shows the title the plugin declares, by the plugin', () => {
    expect(getFileMenuText(recognise)).toBe('Read the scan with Example OCR')
    expect(getFileMenuText(deleteVersion)).toBe(WORDS.deleteVersionOf('Example OCR'))
  })

  it('is offered only over the loaded file', () => {
    const [one] = createFileMenuPaletteCommands('books/Scan.pdf', [recognise])

    expect(one?.group).toBe('file')
    expect(one?.isOffered(front('books/Scan.pdf'), runs)).toBe(true)
    expect(one?.isOffered(front('books/Other.pdf'), runs)).toBe(false)
    expect(one?.isOffered(front('books/Scan.pdf', false), runs)).toBe(false)
    expect(createFileMenuPaletteCommands('', [recognise])[0]?.isOffered(front(''), runs)).toBe(
      false,
    )
  })

  it('draws the icon the plugin names, and a deletion as a deletion', () => {
    const [made] = createFileMenuPaletteCommands('books/Scan.pdf', [recognise])
    expect(made?.icon).toBe(getNamedIcon('scan-text'))
    expect(made?.icon).not.toBe(getNamedIcon('no-such-icon'))
    expect(findIcon(getFileMenuId(deleteVersion))).not.toBeNull()
  })
})

describe('useFileMenuCommands', () => {
  it('loads the menu of the file and reports a started run', async () => {
    const get = vi.fn().mockResolvedValue([recognise])
    const run = vi.fn().mockResolvedValue(undefined)
    const menu = useFileMenuCommands({ get, run })
    await menu.load('books/Scan.pdf')
    const writeMessage = vi.fn()

    await runInvocation(
      invoke(recognise, 'books/Scan.pdf'),
      { writeMessage } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(get).toHaveBeenCalledWith('books/Scan.pdf')
    expect(run).toHaveBeenCalledWith('books/Scan.pdf', recognise)
    expect(writeMessage).toHaveBeenLastCalledWith(WORDS.textAsked('Example OCR'), 'report')
    expect(menu.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      getFileMenuId(recognise),
    ])
  })

  it('reports a failed run as an error', async () => {
    const menu = useFileMenuCommands({
      get: async () => [deleteVersion],
      run: async () => {
        throw new Error('the plugin did not answer')
      },
    })
    await menu.load('books/Scan.pdf')
    const writeMessage = vi.fn()

    await commands.get(getFileMenuId(deleteVersion))?.(
      invoke(deleteVersion, 'books/Scan.pdf'),
      { writeMessage } as unknown as CommandDeps,
      ANSWER_WORDS,
    )

    expect(writeMessage).toHaveBeenLastCalledWith(
      expect.stringContaining('not responding'),
      'error',
    )
  })

  it('clears the entries for an empty file', async () => {
    const get = vi.fn().mockResolvedValue([recognise])
    const menu = useFileMenuCommands({ get, run: vi.fn() })
    await menu.load('books/Scan.pdf')

    await menu.load('')

    expect(get).toHaveBeenCalledTimes(1)
    expect(menu.paletteCommands.value).toStrictEqual([])
    expect(commands.get(getFileMenuId(recognise))).toBeUndefined()
  })
})
