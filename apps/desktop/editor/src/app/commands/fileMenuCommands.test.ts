/**
 * The menu the text versions of the file in front make, as the palette offers
 * it, and an entry run as the palette carries it out.
 */
import { describe, expect, it, vi } from 'vitest'
import { commands, runInvocation, ANSWER_WORDS, type CommandDeps } from '@/features/command-palette'
import type { FileMenuEntry } from '@/entities/text-version'
import { WORDS } from '@/shared/words'
import { findIcon } from '@/shared/icons'
import {
  createFileMenuPaletteCommands,
  getFileMenuId,
  getFileMenuText,
  useFileMenuCommands,
} from './fileMenuCommands'

const recognise: FileMenuEntry = {
  action: 'recognise',
  pluginId: 'md.numen.engines.ocr',
  pluginName: 'Document OCR',
  versionId: '',
}
const deleteVersion: FileMenuEntry = {
  action: 'delete-version',
  pluginId: 'md.numen.engines.ocr',
  pluginName: 'Document OCR',
  versionId: 'md.numen.engines.ocr/recognition',
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
  it('names each entry by action and plugin', () => {
    expect(getFileMenuText(recognise)).toBe(WORDS.recogniseWith('Document OCR'))
    expect(getFileMenuText({ ...recognise, action: 'proofread' })).toBe(
      WORDS.proofreadWith('Document OCR'),
    )
    expect(getFileMenuText(deleteVersion)).toBe(WORDS.deleteVersionOf('Document OCR'))
    expect(getFileMenuText(recognise)).toMatch(/^Recognise/)
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

  it('uses the icon of the action', () => {
    expect(findIcon(getFileMenuId(recognise))).not.toBeNull()
    expect(findIcon(getFileMenuId(deleteVersion))).not.toBeNull()
    expect(findIcon(getFileMenuId(recognise))).not.toBe(findIcon(getFileMenuId(deleteVersion)))
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
    expect(writeMessage).toHaveBeenLastCalledWith(WORDS.textAsked('Document OCR'), 'report')
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
      expect.stringContaining('did not answer'),
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
