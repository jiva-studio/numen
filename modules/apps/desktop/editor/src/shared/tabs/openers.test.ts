/**
 * The one place a file of the vault is opened from, asked without a window.
 *
 * A road holds a path and no choice, so what is asked here is that the path
 * alone decides the editor: the vault is asked what stands at it, and the kind
 * it answers picks the tab the file opens in.
 */
import { describe, expect, it } from 'vitest'
import type { FileKind, RefusalReason } from '../core'
import { fileMakers, fileOpeners, type VaultMaker, type FileOpenerDeps } from './openers'
import { writer } from '../testing/writer'
import { REFUSED } from '../words'

/** A vault that answers what it was told, and counts the questions. */
const vault = (stands: Record<string, FileKind> = {}) => {
  const asked: (readonly string[])[] = []
  const core: FileOpenerDeps = {
    fileKinds: async (paths) => {
      asked.push(paths)
      const found = new Map<string, FileKind>()
      for (const path of paths) {
        const one = stands[path]
        if (one) found.set(path, one)
      }
      return found
    },
  }
  return { core, asked }
}

/** A note of one of three, as the vault answers what stands at its path. */
const note = (type: FileKind['type']): FileKind => ({ kind: 'note', type })

/**
 * A book drawn as pictures of its pages, a book that reflows, a recording, and
 * a file the vault holds no source for.
 */
const BOOK: FileKind = { kind: 'book', type: 'note', format: 'pdf' }
const REFLOWS: FileKind = { kind: 'book', type: 'note', format: 'epub' }
const TALK: FileKind = { kind: 'recording', type: 'note' }
const OTHER: FileKind = { kind: 'other', type: 'note' }

/** A vault that cannot answer at all. */
const unreachable: FileOpenerDeps = {
  fileKinds: async () => {
    throw new Error('the vault is not there')
  },
}

/** The three editors, the reader and the player, each writing down what it was given. */
const editors = (puts: ReturnType<typeof fileOpeners>) => {
  const opened: string[] = []
  for (const type of ['note', 'deck', 'stencil'] as const) {
    puts.holds(type, (path, title, showing, line) =>
      opened.push(`${type} ${path} ${title || '—'} ${showing} ${line ?? '—'}`),
    )
  }
  puts.reads((path, runs) =>
    opened.push(
      `document ${path} [${runs.map((one) => `${one.from}+${one.to}`).join(', ')}]`,
    ),
  )
  puts.turns((path, runs) =>
    opened.push(`book ${path} [${runs.map((one) => `${one.from}+${one.to}`).join(', ')}]`),
  )
  puts.hears((path, runs) =>
    opened.push(
      `recording ${path} [${runs.map((one) => `${one.from}+${one.to}`).join(', ')}]`,
    ),
  )
  return opened
}

describe('a path opened', () => {
  it('opens a deck in the editor of its cards', async () => {
    const one = vault({ 'Animals.md': note('deck') })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Animals.md', 'Animals')

    expect(opened).toStrictEqual(['deck Animals.md Animals here —'])
  })

  it('opens a stencil in the editor of its fields and faces', async () => {
    const one = vault({ 'Animal.md': note('stencil') })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Animal.md', 'Animal')

    expect(opened).toStrictEqual(['stencil Animal.md Animal here —'])
  })

  it('opens an ordinary note in the editor of its prose', async () => {
    const one = vault({ 'Entropy.md': note('note') })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy')

    expect(opened).toStrictEqual(['note Entropy.md Entropy here —'])
  })

  it('opens a book in the reader, and in no editor of a note', async () => {
    const one = vault({ 'Physics.epub': BOOK })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Physics.epub', 'Physics')

    expect(opened).toStrictEqual(['document Physics.epub []'])
  })

  it('opens a recording in the player, and in no reader', async () => {
    const one = vault({ 'talks/Ants.mp3': TALK })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('talks/Ants.mp3', 'Ants')

    expect(opened).toStrictEqual(['recording talks/Ants.mp3 []'])
  })

  it('opens a file the vault holds no source for in nothing at all', async () => {
    const one = vault({ 'Cover.png': OTHER })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Cover.png', 'Cover')

    expect(opened).toStrictEqual([])
  })

  it('opens nothing where the vault says nothing stands there', async () => {
    const one = vault()
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Gone.md')

    expect(opened).toStrictEqual([])
  })

  it('opens as a note where the vault could not be asked at all', async () => {
    const puts = fileOpeners(unreachable)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy')

    expect(opened).toStrictEqual(['note Entropy.md Entropy here —'])
  })

  it('asks the vault what stands there, the caller having said nothing', async () => {
    const one = vault({ 'Animals.md': note('deck') })
    const puts = fileOpeners(one.core)
    editors(puts)

    await puts.opens('Animals.md')

    expect(one.asked).toStrictEqual([['Animals.md']])
  })

  it('opens beside where that is where it was asked for', async () => {
    const one = vault({ 'Animals.md': note('deck') })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Animals.md', 'Animals', 'beside')

    expect(opened).toStrictEqual(['deck Animals.md Animals beside —'])
  })

  it('carries the line it was asked at, which each editor answers for itself', async () => {
    const one = vault({ 'Entropy.md': note('note'), 'Animals.md': note('deck') })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy', 'here', 12)
    await puts.opens('Animals.md', 'Animals', 'here', 12)

    expect(opened).toStrictEqual([
      'note Entropy.md Entropy here 12',
      'deck Animals.md Animals here 12',
    ])
  })
})

describe('a source opened at a stretch of its own text', () => {
  it('reads a book at the stretches it was asked at', async () => {
    const one = vault({ 'Physics.epub': BOOK })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opensAt('Physics.epub', [
      { from: 10, to: 14 },
      { from: 30, to: 32 },
    ])

    expect(opened).toStrictEqual(['document Physics.epub [10+14, 30+32]'])
  })

  it('plays a recording at the stretch of the words it was asked at', async () => {
    const one = vault({ 'talks/Ants.mp3': TALK })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opensAt('talks/Ants.mp3', [{ from: 22, to: 28 }])

    expect(opened).toStrictEqual(['recording talks/Ants.mp3 [22+28]'])
  })

  it('opens a note in the editor made for what it is, and not in the reader', async () => {
    const one = vault({ 'Animals.md': note('deck'), 'Entropy.md': note('note') })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opensAt('Animals.md', [{ from: 10, to: 14 }])
    await puts.opensAt('Entropy.md', [{ from: 10, to: 14 }])

    expect(opened).toStrictEqual(['deck Animals.md — here —', 'note Entropy.md — here —'])
  })

  it('opens nothing where the vault says nothing stands there', async () => {
    const one = vault()
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opensAt('Gone.epub', [{ from: 10, to: 14 }])

    expect(opened).toStrictEqual([])
  })
})

describe('which reader a book opens in', () => {
  it('turns a book that reflows a spread at a time', async () => {
    const one = vault({ 'Gita.epub': REFLOWS })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opensAt('Gita.epub', [{ from: 10, to: 14 }])

    expect(opened).toStrictEqual(['book Gita.epub [10+14]'])
  })

  it('reads a book of pages as the row of pages it is', async () => {
    const one = vault({ 'Physics.pdf': BOOK })
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    await puts.opens('Physics.pdf')

    expect(opened).toStrictEqual(['document Physics.pdf []'])
  })

  it('reads a book that reflows as pages where the window turns none', async () => {
    // A window told about no book tab still opens the file.
    const one = vault({ 'Gita.epub': REFLOWS })
    const puts = fileOpeners(one.core)
    const opened: string[] = []
    puts.reads((path) => opened.push(`document ${path}`))

    await puts.opens('Gita.epub')

    expect(opened).toStrictEqual(['document Gita.epub'])
  })
})

describe('a file just made here', () => {
  it('opens as what it was made as, the vault not being asked', async () => {
    const one = vault()
    const puts = fileOpeners(one.core)
    const opened = editors(puts)

    puts.made('Animals.md', 'Animals', 'deck')

    expect(opened).toStrictEqual(['deck Animals.md Animals here —'])
    expect(one.asked).toStrictEqual([])
  })
})

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
    makesPreset: (title, folder) => {
      asked.push(`preset ${folder || '—'} ${title}`)
      return answer(`${folder}/${title}.md`)
    },
    makesURL: (address, folder) => {
      asked.push(`link ${folder || '—'} ${address}`)
      return answer(`${folder}/${address}.md`)
    },
  }
}

/** Everything making one of the four says, and the field a stencil carries. */
const MAKING = { refused: REFUSED, field: 'Front' }

describe('a deck, a stencil or a preset made', () => {
  it('is asked of the vault under the name and the folder it was given', async () => {
    const vault = maker()
    const made = fileMakers(vault, fileOpeners(unreachable), MAKING, () => {})

    await made.makes('deck', 'zoology', 'Animals')
    await made.makes('stencil', 'zoology', 'Words')
    await made.makes('preset', '', 'Slow')

    expect(vault.asked).toStrictEqual([
      'deck zoology Animals',
      'stencil zoology Words Front',
      'preset — Slow',
    ])
  })

  it('answers where the vault filed it', async () => {
    const made = fileMakers(maker(), fileOpeners(unreachable), MAKING, () => {})

    expect(await made.makes('deck', 'zoology', 'Animals')).toBe('zoology/Animals.md')
  })

  it('is put in front of the person as what it was made as', async () => {
    const puts = fileOpeners(unreachable)
    const opened = editors(puts)
    puts.holds('preset', (path) => opened.push(`preset ${path}`))
    const made = fileMakers(maker(), puts, MAKING, () => {})

    await made.decks('zoology', 'Animals')
    await made.stencils('zoology', 'Words')
    await made.presets('', 'Slow')

    expect(opened).toStrictEqual([
      'deck zoology/Animals.md — here —',
      'stencil zoology/Words.md — here —',
      'preset /Slow.md',
    ])
  })

  it('says what the vault refused, and nothing opens', async () => {
    const puts = fileOpeners(unreachable)
    const opened = editors(puts)
    const told = writer()
    const made = fileMakers(maker('occupied'), puts, MAKING, told.says)

    expect(await made.decks('zoology', 'Animals')).toBe('')
    expect(told.said).toStrictEqual([REFUSED.occupied])
    expect(opened).toStrictEqual([])
  })

  it('says a vault that could not be asked at all', async () => {
    const told = writer()
    const made = fileMakers(maker(null, true), fileOpeners(unreachable), MAKING, told.says)

    expect(await made.decks('zoology', 'Animals')).toBe('')
    expect(told.said.join(' ')).not.toContain('the vault is not there')
    expect(told.said.join(' ')).toContain('numen did not answer')
  })
})
