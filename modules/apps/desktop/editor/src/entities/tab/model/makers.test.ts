/**
 * What making one of the four asks of the vault, and what it says when the
 * vault refuses or cannot be asked at all.
 */
import { describe, expect, it } from 'vitest'
import type { ErrorCode } from '@/shared/errors'
import type { CreateResult } from '@/entities/file/@x/tab'
import { writer } from '@/testing/writer'
import { ERRORS } from '@/shared/words'
import { fileOpeners } from './openers'
import { createFileCreators, type VaultCreator } from './makers'

/** The vault answering what it was told, and writing down what it was asked to make. */
const maker = (
  error: ErrorCode | null = null,
  throws = false,
): VaultCreator & { asked: string[] } => {
  const asked: string[] = []
  const answer = async (path: string): Promise<CreateResult> => {
    if (throws) throw new Error('the vault is not there')
    return error ? { ok: false, error } : { ok: true, value: { path } }
  }
  return {
    asked,
    createDeck: (title, folder) => {
      asked.push(`deck ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    createStencil: (title, folder, fields) => {
      asked.push(`stencil ${folder || '—'} ${title} ${fields.join(',')}`)
      return answer(`${folder}/${title}.md`)
    },
    createPreset: (title, folder) => {
      asked.push(`preset ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    createUrl: (address, folder) => {
      asked.push(`link ${folder || '—'} ${address}`)
      return answer(`${folder}/${address}.md`)
    },
  }
}

/** Everything making one of the four says. */
const MAKING = { errors: ERRORS }

const createCreators = (
  vault: VaultCreator,
  tabOpeners = fileOpeners({ fileKinds: async () => new Map() }),
  write = () => {},
) => createFileCreators(vault, tabOpeners, { words: MAKING, write })

describe('a deck, a stencil or a preset made', () => {
  it('is asked of the vault under the name and the folder it was given', async () => {
    const vault = maker()
    const made = createCreators(vault)

    await made.createFile('deck', 'zoology', 'Animals')
    await made.createFile('stencil', 'zoology', { name: 'Words', fields: ['Front'] })
    await made.createFile('preset', '', 'Slow')

    expect(vault.asked).toStrictEqual([
      'deck zoology Animals',
      'stencil zoology Words Front',
      'preset — Slow',
    ])
  })

  it('answers where the vault filed it', async () => {
    const made = createCreators(maker())

    expect(await made.createFile('deck', 'zoology', 'Animals')).toBe('zoology/Animals.md')
  })

  it('is put in front of the person as what it was made as', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const opened: string[] = []
    for (const what of ['deck', 'stencil', 'preset'] as const) {
      tabOpeners.registerEditor(what, (path) => opened.push(`${what} ${path}`))
    }
    const made = createCreators(maker(), tabOpeners)

    await made.createDeck('zoology', 'Animals')
    await made.createStencil('zoology', 'Words', ['Front'])
    await made.createPreset('', 'Slow')

    expect(opened).toStrictEqual([
      'deck zoology/Animals.md',
      'stencil zoology/Words.md',
      'preset /Slow.md',
    ])
  })

  it('says what the vault refused, and nothing opens', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const told = writer()
    const made = createCreators(maker('occupied'), tabOpeners, told.write)

    expect(await made.createDeck('zoology', 'Animals')).toBe('')
    expect(told.said).toStrictEqual([ERRORS.occupied])
  })

  it('says a vault that could not be asked at all', async () => {
    const told = writer()
    const made = createCreators(maker(null, true), undefined, told.write)

    expect(await made.createDeck('zoology', 'Animals')).toBe('')
    expect(told.said.join(' ')).not.toContain('the vault is not there')
    expect(told.said.join(' ')).toContain('numen did not answer')
  })

  it('is asked of the vault with no fields where none were given', async () => {
    const vault = maker()
    const made = createCreators(vault)

    await made.createFile('stencil', 'zoology', 'Words')

    expect(vault.asked).toStrictEqual(['stencil zoology Words '])
  })

  it('says what the vault refused as an error', async () => {
    const told = writer()
    const made = createCreators(maker('occupied'), undefined, told.write)

    await made.createFile('deck', 'zoology', 'Animals')

    expect(told.told).toStrictEqual([{ text: ERRORS.occupied, kind: 'error' }])
  })

  it('says a vault that could not be asked as an error', async () => {
    const told = writer()
    const made = createCreators(maker(null, true), undefined, told.write)

    await made.createFile('deck', 'zoology', 'Animals')

    expect(told.told.map((one) => one.kind)).toStrictEqual(['error'])
  })
})

describe('a file made and put in front of the person', () => {
  it('is opened with no title of its own', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const titles: string[] = []
    tabOpeners.registerEditor('deck', (_path, title) => titles.push(title))
    const made = createCreators(maker(), tabOpeners)

    await made.createDeck('zoology', 'Animals')

    expect(titles).toStrictEqual([''])
  })
})

describe('a note pointing at an address made', () => {
  it('is asked of the vault under the address and folder, and put in front of the person', async () => {
    const vault = maker()
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const opened: string[] = []
    tabOpeners.registerEditor('url', (path) => opened.push(path))
    const made = createCreators(vault, tabOpeners)

    const path = await made.createUrl('links', 'example.org')

    expect(vault.asked).toStrictEqual(['link links example.org'])
    expect(path).toBe('links/example.org.md')
    expect(opened).toStrictEqual(['links/example.org.md'])
  })

  it('answers nothing, and opens nothing, where the vault refused', async () => {
    const tabOpeners = fileOpeners({ fileKinds: async () => new Map() })
    const opened: string[] = []
    tabOpeners.registerEditor('url', (path) => opened.push(path))
    const made = createCreators(maker('occupied'), tabOpeners)

    expect(await made.createUrl('links', 'example.org')).toBe('')
    expect(opened).toStrictEqual([])
  })
})
