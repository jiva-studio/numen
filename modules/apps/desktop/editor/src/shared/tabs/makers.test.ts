/**
 * What making one of the four asks of the vault, and what it says when the
 * vault refuses or cannot be asked at all.
 */
import { describe, expect, it } from 'vitest'
import type { RefusalReason } from '../note'
import { writer } from '../testing/writer'
import { REFUSED } from '../words'
import { fileOpeners } from './openers'
import { fileMakers, type VaultMaker } from './makers'

/** The vault answering what it was told, and writing down what it was asked to make. */
const maker = (
  refusal: RefusalReason | null = null,
  throws = false,
): VaultMaker & { asked: string[] } => {
  const asked: string[] = []
  const answer = async (path: string) => {
    if (throws) throw new Error('the vault is not there')
    return { path: refusal ? '' : path, refusal }
  }
  return {
    asked,
    makeDeck: (title, folder) => {
      asked.push(`deck ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    makeStencil: (title, folder, fields) => {
      asked.push(`stencil ${folder || '—'} ${title} ${fields.join(',')}`)
      return answer(`${folder}/${title}.md`)
    },
    makePreset: (title, folder) => {
      asked.push(`preset ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    makeURL: (address, folder) => {
      asked.push(`link ${folder || '—'} ${address}`)
      return answer(`${folder}/${address}.md`)
    },
  }
}

/** Everything making one of the four says. */
const MAKING = { refused: REFUSED }

describe('a deck, a stencil or a preset made', () => {
  it('is asked of the vault under the name and the folder it was given', async () => {
    const vault = maker()
    const made = fileMakers(vault, fileOpeners({ fileKinds: async () => new Map() }), MAKING, () => {})

    await made.makes('deck', 'zoology', 'Animals')
    await made.makes('stencil', 'zoology', 'Words', ['Front'])
    await made.makes('preset', '', 'Slow')

    expect(vault.asked).toStrictEqual([
      'deck zoology Animals',
      'stencil zoology Words Front',
      'preset — Slow',
    ])
  })

  it('answers where the vault filed it', async () => {
    const made = fileMakers(maker(), fileOpeners({ fileKinds: async () => new Map() }), MAKING, () => {})

    expect(await made.makes('deck', 'zoology', 'Animals')).toBe('zoology/Animals.md')
  })

  it('is put in front of the person as what it was made as', async () => {
    const puts = fileOpeners({ fileKinds: async () => new Map() })
    const opened: string[] = []
    for (const what of ['deck', 'stencil', 'preset'] as const) {
      puts.holds(what, (path) => opened.push(`${what} ${path}`))
    }
    const made = fileMakers(maker(), puts, MAKING, () => {})

    await made.decks('zoology', 'Animals')
    await made.stencils('zoology', 'Words', ['Front'])
    await made.presets('', 'Slow')

    expect(opened).toStrictEqual([
      'deck zoology/Animals.md',
      'stencil zoology/Words.md',
      'preset /Slow.md',
    ])
  })

  it('says what the vault refused, and nothing opens', async () => {
    const puts = fileOpeners({ fileKinds: async () => new Map() })
    const told = writer()
    const made = fileMakers(maker('occupied'), puts, MAKING, told.says)

    expect(await made.decks('zoology', 'Animals')).toBe('')
    expect(told.said).toStrictEqual([REFUSED.occupied])
  })

  it('says a vault that could not be asked at all', async () => {
    const told = writer()
    const made = fileMakers(maker(null, true), fileOpeners({ fileKinds: async () => new Map() }), MAKING, told.says)

    expect(await made.decks('zoology', 'Animals')).toBe('')
    expect(told.said.join(' ')).not.toContain('the vault is not there')
    expect(told.said.join(' ')).toContain('numen did not answer')
  })
})
