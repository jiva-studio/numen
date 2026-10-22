/**
 * What making one of the four asks of the vault, and what it says when the
 * vault refuses or cannot be asked at all.
 */
import { describe, expect, it } from 'vitest'
import type { ErrorCode } from '@/shared/errors'
import type { CreateResult } from '@/entities/file/@x/tab'
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
    createDeck: (title, folder) => {
      request.push(`deck ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    createStencil: (title, folder) => {
      request.push(`stencil ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    createPreset: (title, folder) => {
      request.push(`preset ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    createUrl: (address, folder) => {
      request.push(`link ${folder || '—'} ${address}`)
      return answer(`${folder}/${address}.md`)
    },
  }
}

/** Everything making one of the four says. */
const MAKING = { errors: ERRORS }

const createCreators = (
  vault: VaultCreator,
  tabOpeners = fileOpeners({ fileKinds: async () => new Map() }),
  write: MessageWriter = () => {},
) => createFileCreators(vault, tabOpeners, { words: MAKING, write })

describe('a deck, a stencil or a preset made', () => {
  it('is requested of the vault under the name and the folder it was given', async () => {
    const vault = maker()
    const createdItem = createCreators(vault)

    await createdItem.createFile('deck', 'zoology', 'Animals')
    await createdItem.createFile('stencil', 'zoology', 'Words')
    await createdItem.createFile('preset', '', 'Slow')

    expect(vault.requests).toStrictEqual([
      'deck zoology Animals',
      'stencil zoology Words',
      'preset — Slow',
    ])
  })

  it('answers where the vault filed it', async () => {
    const createdItem = createCreators(maker())

    expect(await createdItem.createFile('deck', 'zoology', 'Animals')).toBe('zoology/Animals.md')
  })

  it('is put in front of the person as what it was made as', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    for (const what of ['deck', 'stencil', 'preset'] as const) {
      tabOpeners.registerEditor(what, (path) => openedTabs.push(`${what} ${path}`))
    }
    const createdItem = createCreators(maker(), tabOpeners)

    await createdItem.createDeck('zoology', 'Animals')
    await createdItem.createStencil('zoology', 'Words')
    await createdItem.createPreset('', 'Slow')

    expect(openedTabs).toStrictEqual([
      'deck zoology/Animals.md',
      'stencil zoology/Words.md',
      'preset /Slow.md',
    ])
  })

  it('says what the vault rejected, and nothing opens', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const mockWriter = writer()
    const createdItem = createCreators(maker('occupied'), tabOpeners, mockWriter.write)

    expect(await createdItem.createDeck('zoology', 'Animals')).toBe('')
    expect(mockWriter.messagesText).toStrictEqual([ERRORS.occupied])
  })

  it('says a vault that could not be requested at all', async () => {
    const mockWriter = writer()
    const createdItem = createCreators(maker(null, true), undefined, mockWriter.write)

    expect(await createdItem.createDeck('zoology', 'Animals')).toBe('')
    expect(mockWriter.messagesText.join(' ')).not.toContain('the vault is not there')
    expect(mockWriter.messagesText.join(' ')).toContain('numen did not answer')
  })

  it('says what the vault rejected as an error', async () => {
    const mockWriter = writer()
    const createdItem = createCreators(maker('occupied'), undefined, mockWriter.write)

    await createdItem.createFile('deck', 'zoology', 'Animals')

    expect(mockWriter.messages).toStrictEqual([{ text: ERRORS.occupied, kind: 'error' }])
  })

  it('says a vault that could not be requested as an error', async () => {
    const mockWriter = writer()
    const createdItem = createCreators(maker(null, true), undefined, mockWriter.write)

    await createdItem.createFile('deck', 'zoology', 'Animals')

    expect(mockWriter.messages.map((one) => one.kind)).toStrictEqual(['error'])
  })
})

describe('a file nothing offers to make', () => {
  it('answers nothing, opens nothing and writes nothing of its own, the creator having returned why', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    tabOpeners.registerEditor('deck', (path) => openedTabs.push(path))
    const mockWriter = writer()
    const createdItem = createCreators(
      { createDeck: async () => null },
      tabOpeners,
      mockWriter.write,
    )

    expect(await createdItem.createDeck('zoology', 'Animals')).toBe('')
    expect(openedTabs).toStrictEqual([])
    expect(mockWriter.messagesText).toStrictEqual([])
  })
})

describe('a file made and put in front of the person', () => {
  it('is opened with no title of its own', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const titles: string[] = []
    tabOpeners.registerEditor('deck', (_path, title) => titles.push(title))
    const createdItem = createCreators(maker(), tabOpeners)

    await createdItem.createDeck('zoology', 'Animals')

    expect(titles).toStrictEqual([''])
  })
})

describe('a note pointing at an address made', () => {
  it('is requested of the vault under the address and folder, and put in front of the person', async () => {
    const vault = maker()
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    tabOpeners.registerEditor('url', (path) => openedTabs.push(path))
    const createdItem = createCreators(vault, tabOpeners)

    const path = await createdItem.createUrl('links', 'example.org')

    expect(vault.requests).toStrictEqual(['link links example.org'])
    expect(path).toBe('links/example.org.md')
    expect(openedTabs).toStrictEqual(['links/example.org.md'])
  })

  it('answers nothing, and opens nothing, where the vault rejected', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    tabOpeners.registerEditor('url', (path) => openedTabs.push(path))
    const createdItem = createCreators(maker('occupied'), tabOpeners)

    expect(await createdItem.createUrl('links', 'example.org')).toBe('')
    expect(openedTabs).toStrictEqual([])
  })
})

describe('several notes pointing at addresses made', () => {
  it('opens the single file when given one address', async () => {
    const vault = maker()
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    tabOpeners.registerEditor('url', (path) => openedTabs.push(path))
    const createdItem = createCreators(vault, tabOpeners)

    const paths = await createdItem.createUrls('links', ['example.org'])

    expect(vault.requests).toStrictEqual(['link links example.org'])
    expect(paths).toStrictEqual(['links/example.org.md'])
    expect(openedTabs).toStrictEqual(['links/example.org.md'])
  })

  it('creates every note and opens none when given several addresses', async () => {
    const vault = maker()
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const openedTabs: string[] = []
    tabOpeners.registerEditor('url', (path) => openedTabs.push(path))
    const createdItem = createCreators(vault, tabOpeners)

    const paths = await createdItem.createUrls('links', ['first.org', 'second.org'])

    expect(vault.requests).toStrictEqual(['link links first.org', 'link links second.org'])
    expect(paths).toStrictEqual(['links/first.org.md', 'links/second.org.md'])
    expect(openedTabs).toStrictEqual([])
  })

  it('answers nothing when given an empty list', async () => {
    const vault = maker()
    const createdItem = createCreators(vault)

    const paths = await createdItem.createUrls('links', [])

    expect(vault.requests).toStrictEqual([])
    expect(paths).toStrictEqual([])
  })
})
