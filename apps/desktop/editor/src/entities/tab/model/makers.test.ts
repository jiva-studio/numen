/**
 * What making a file of a kind asks of the vault, and what it says when the vault
 * refuses or cannot be asked at all.
 */
import { describe, expect, it } from 'vitest'
import type { ErrorCode } from '@/shared/errors'
import type { CreateResult } from '@/shared/model/fileTypes'
import { writer } from '@/testing/writer'
import { ERRORS } from '@/shared/words'
import type { MessageWriter } from '@/shared/notices/messages'
import { fileOpeners } from './openers'
import { createFileCreators, type VaultCreator } from './makers'

/** The vault answering what it was told, and writing down what it was asked to make. */
const maker = (
  error: ErrorCode | null = null,
  throws = false,
): VaultCreator & { requests: string[] } => {
  const request: string[] = []
  const answer = async (path: string): Promise<CreateResult> => {
    if (throws) throw new Error('the vault is not there')
    return error ? { ok: false, error } : { ok: true, value: { path } }
  }
  return {
    requests: request,
    createFile: (kind, title, folder) => {
      request.push(`${kind} ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
  }
}

/** Everything making a file says. */
const MAKING = { errors: ERRORS }

const createCreators = (
  vault: VaultCreator,
  tabOpeners = fileOpeners({ fileKinds: async () => new Map() }),
  write: MessageWriter = () => {},
) => createFileCreators(vault, tabOpeners, { words: MAKING, write })

describe('a file of a kind made', () => {
  it('is requested of the vault under the name and the folder it was given', async () => {
    const vault = maker()
    const createdItem = createCreators(vault)

    await createdItem.createFile('sample', 'zoology', 'Animals')
    await createdItem.createFile('form', 'zoology', 'Words')
    await createdItem.createFile('sheet', '', 'Slow')

    expect(vault.requests).toStrictEqual([
      'sample zoology Animals',
      'form zoology Words',
      'sheet — Slow',
    ])
  })

  it('answers where the vault filed it', async () => {
    const createdItem = createCreators(maker())

    expect(await createdItem.createFile('sample', 'zoology', 'Animals')).toBe('zoology/Animals.md')
  })

  it('is put in front of the person as what it was made as', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    for (const what of ['sample', 'form', 'sheet'] as const) {
      tabOpeners.registerEditor(what, (path) => openedTabs.push(`${what} ${path}`))
    }
    const createdItem = createCreators(maker(), tabOpeners)

    await createdItem.createAndOpenFile('sample', 'zoology', 'Animals')
    await createdItem.createAndOpenFile('form', 'zoology', 'Words')
    await createdItem.createAndOpenFile('sheet', '', 'Slow')

    expect(openedTabs).toStrictEqual([
      'sample zoology/Animals.md',
      'form zoology/Words.md',
      'sheet /Slow.md',
    ])
  })

  it('says what the vault rejected, and nothing opens', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const mockWriter = writer()
    const createdItem = createCreators(maker('alreadyExists'), tabOpeners, mockWriter.write)

    expect(await createdItem.createAndOpenFile('sample', 'zoology', 'Animals')).toBe('')
    expect(mockWriter.messagesText).toStrictEqual([ERRORS.alreadyExists])
  })

  it('says a vault that could not be requested at all', async () => {
    const mockWriter = writer()
    const createdItem = createCreators(maker(null, true), undefined, mockWriter.write)

    expect(await createdItem.createAndOpenFile('sample', 'zoology', 'Animals')).toBe('')
    expect(mockWriter.messagesText.join(' ')).not.toContain('the vault is not there')
    expect(mockWriter.messagesText.join(' ')).toContain('Numen is not responding')
  })

  it('says what the vault rejected as an error', async () => {
    const mockWriter = writer()
    const createdItem = createCreators(maker('alreadyExists'), undefined, mockWriter.write)

    await createdItem.createFile('sample', 'zoology', 'Animals')

    expect(mockWriter.messages).toStrictEqual([{ text: ERRORS.alreadyExists, kind: 'error' }])
  })

  it('says a vault that could not be requested as an error', async () => {
    const mockWriter = writer()
    const createdItem = createCreators(maker(null, true), undefined, mockWriter.write)

    await createdItem.createFile('sample', 'zoology', 'Animals')

    expect(mockWriter.messages.map((one) => one.kind)).toStrictEqual(['error'])
  })
})

describe('a file nothing offers to make', () => {
  it('returns an empty path, opens no tab and writes no message when the creator fails', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    tabOpeners.registerEditor('sample', (path) => openedTabs.push(path))
    const mockWriter = writer()
    const createdItem = createCreators(
      { createFile: async () => null },
      tabOpeners,
      mockWriter.write,
    )

    expect(await createdItem.createAndOpenFile('sample', 'zoology', 'Animals')).toBe('')
    expect(openedTabs).toStrictEqual([])
    expect(mockWriter.messagesText).toStrictEqual([])
  })
})

describe('a file made and put in front of the person', () => {
  it('is opened with no title of its own', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const titles: string[] = []
    tabOpeners.registerEditor('sample', (_path, title) => titles.push(title))
    const createdItem = createCreators(maker(), tabOpeners)

    await createdItem.createAndOpenFile('sample', 'zoology', 'Animals')

    expect(titles).toStrictEqual([''])
  })
})
