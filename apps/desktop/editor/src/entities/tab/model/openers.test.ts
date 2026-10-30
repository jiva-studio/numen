/**
 * The one place a file of the vault is opened from, asked without a window.
 *
 * A note opens in the editor of its type; any other file in the reader
 * registered for its extension, at the selector given.
 */
import { describe, expect, it, vi } from 'vitest'
import type { FileKind } from '@/shared/model/fileTypes'
import { fileOpeners, type FileOpenerDeps } from './openers'

/** A vault that answers what it was told, and counts the questions. */
const vault = (stands: Record<string, FileKind> = {}) => {
  const request: (readonly string[])[] = []
  const core: FileOpenerDeps = {
    fileKinds: async (paths) => {
      request.push(paths)
      const found = new Map<string, FileKind>()
      for (const path of paths) {
        const one = stands[path]
        if (one) found.set(path, one)
      }
      return found
    },
  }
  return { core, requested: request }
}

const note = (type: FileKind['type']): FileKind => ({ isNote: true, type })

const OTHER: FileKind = { isNote: false, type: 'note' }

/** A vault that cannot answer at all. */
const unreachable: FileOpenerDeps = {
  fileKinds: async () => {
    throw new Error('the vault is not there')
  },
}

/** The editors of three note types and a reader of `.bin`, each writing down what it was given. */
const editors = (tabOpeners: ReturnType<typeof fileOpeners>) => {
  const activeEditors: string[] = []
  for (const type of ['note', 'sample', 'form'] as const) {
    tabOpeners.registerEditor(type, (path, title, options) => {
      const showing = typeof options === 'string' ? options : (options?.where ?? 'current')
      const line = typeof options === 'object' ? options?.line : undefined
      activeEditors.push(`${type} ${path} ${title || '—'} ${showing} ${line ?? '—'}`)
    })
  }
  tabOpeners.registerExtensionReader('bin', (path, selector) =>
    activeEditors.push(`reader ${path} ${JSON.stringify(selector)}`),
  )
  return activeEditors
}

describe('a path opened', () => {
  it('opens a sample in the editor of its kind', async () => {
    const one = vault({ 'Animals.md': note('sample') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Animals.md', 'Animals')

    expect(activeEditors).toStrictEqual(['sample Animals.md Animals current —'])
  })

  it('opens a form in the editor of its kind', async () => {
    const one = vault({ 'Animal.md': note('form') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Animal.md', 'Animal')

    expect(activeEditors).toStrictEqual(['form Animal.md Animal current —'])
  })

  it('opens an ordinary note in the editor of its prose', async () => {
    const one = vault({ 'Entropy.md': note('note') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Entropy.md', 'Entropy')

    expect(activeEditors).toStrictEqual(['note Entropy.md Entropy current —'])
  })

  it('opens a file that is no note in the reader of its extension, at no place', async () => {
    const one = vault({ 'Physics.bin': OTHER })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Physics.bin', 'Physics')

    expect(activeEditors).toStrictEqual(['reader Physics.bin null'])
  })

  it('opens a file in its reader at the selector requested', async () => {
    const one = vault({ 'Physics.BIN': OTHER })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Physics.BIN', '', { selector: { page: 3 } })

    expect(activeEditors).toStrictEqual(['reader Physics.BIN {"page":3}'])
  })

  it('opens a file no reader takes in nothing at all', async () => {
    const one = vault({ 'Cover.png': OTHER })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Cover.png')

    expect(activeEditors).toStrictEqual([])
  })

  it('opens nothing where the vault says nothing remains there', async () => {
    const one = vault()
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Gone.md')

    expect(activeEditors).toStrictEqual([])
  })

  it('opens as a note where the vault could not be requested at all', async () => {
    const tabOpeners = fileOpeners(unreachable)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Entropy.md', 'Entropy')

    expect(activeEditors).toStrictEqual(['note Entropy.md Entropy current —'])
  })

  it('asks the vault what remains there, the caller having returned nothing', async () => {
    const one = vault({ 'Animals.md': note('sample') })
    const tabOpeners = fileOpeners(one.core)
    editors(tabOpeners)

    await tabOpeners.openFile('Animals.md')

    expect(one.requested).toStrictEqual([['Animals.md']])
  })

  it('passes an editor an empty title where the caller gave none', async () => {
    const one = vault({ 'Animals.md': note('sample') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Animals.md')

    expect(activeEditors).toStrictEqual(['sample Animals.md — current —'])
  })

  it('opens nothing where no editor was handed over for the kind', async () => {
    const one = vault({ 'Animals.md': note('sample') })
    const tabOpeners = fileOpeners(one.core)

    await expect(tabOpeners.openFile('Animals.md', 'Animals')).resolves.toBeUndefined()
  })

  it('opens a file in nothing where no reader was handed over for its extension', async () => {
    const one = vault({ 'talks/Ants.bin': OTHER })
    const tabOpeners = fileOpeners(one.core)

    await expect(tabOpeners.openFile('talks/Ants.bin')).resolves.toBeUndefined()
    await expect(
      tabOpeners.openFileAt('talks/Ants.bin', [{ start: 22, end: 28 }]),
    ).resolves.toBeUndefined()
  })

  it('opens alongside where that is where it was requested for', async () => {
    const one = vault({ 'Animals.md': note('sample') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Animals.md', 'Animals', 'split')

    expect(activeEditors).toStrictEqual(['sample Animals.md Animals split —'])
  })

  it('carries the line it was requested at, which each editor answers for itself', async () => {
    const one = vault({ 'Entropy.md': note('note'), 'Animals.md': note('sample') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('Entropy.md', 'Entropy', { where: 'current', line: 12 })
    await tabOpeners.openFile('Animals.md', 'Animals', { where: 'current', line: 12 })

    expect(activeEditors).toStrictEqual([
      'note Entropy.md Entropy current 12',
      'sample Animals.md Animals current 12',
    ])
  })
})

describe('a file opened at runs of its text', () => {
  it('opens a note in its editor', async () => {
    const one = vault({ 'Entropy.md': note('note') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFileAt('Entropy.md', [{ start: 1, end: 4 }])

    expect(activeEditors).toStrictEqual(['note Entropy.md — current —'])
  })

  it('opens a file that is no note in its reader at no place', async () => {
    const one = vault({ 'Physics.bin': OTHER })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFileAt('Physics.bin', [{ start: 1, end: 4 }])

    expect(activeEditors).toStrictEqual(['reader Physics.bin null'])
  })
})

describe('the icon of a file', () => {
  it('is the icon of the reader of its extension, and nothing where none reads it', () => {
    const tabOpeners = fileOpeners(vault().core)
    const icon = { name: 'Icon' }
    tabOpeners.registerExtensionReader('.bin', vi.fn(), icon)

    expect(tabOpeners.findFileIcon('a/b.BIN')).toBe(icon)
    expect(tabOpeners.findFileIcon('a/b.png')).toBeUndefined()
    expect(tabOpeners.findFileIcon('a.dir/bin')).toBeUndefined()
  })
})

describe('a file just made here', () => {
  it('opens in nothing where no editor was handed over for what it was made as', () => {
    const tabOpeners = fileOpeners(vault().core)

    expect(() => tabOpeners.openNewFile('Animals.md', 'Animals', 'sample')).not.toThrow()
  })

  it('opens as what it was made as, the vault not being requested', async () => {
    const one = vault()
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    tabOpeners.openNewFile('Animals.md', 'Animals', 'sample')

    expect(activeEditors).toStrictEqual(['sample Animals.md Animals current —'])
    expect(one.requested).toStrictEqual([])
  })
})

describe('reconciling open tabs when a file is opened', () => {
  it('reconciles open tabs for a sample file', async () => {
    const reconciled: { path: string; newKind: string }[] = []
    const one = vault({ 'Animals.md': note('sample') })
    const core: FileOpenerDeps = {
      ...one.core,
      reconcileTab: (path, newKind) => reconciled.push({ path, newKind }),
    }
    const tabOpeners = fileOpeners(core)
    editors(tabOpeners)

    await tabOpeners.openFile('Animals.md', 'Animals')

    expect(reconciled).toStrictEqual([{ path: 'Animals.md', newKind: 'sample' }])
  })
})

describe('an extension reader registered twice', () => {
  it('stays registered when the disposer of the reader it replaced runs late', async () => {
    const one = vault({ 'a.xyz': OTHER })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors: string[] = []
    const disposeFirst = tabOpeners.registerExtensionReader('xyz', () =>
      activeEditors.push('first'),
    )
    tabOpeners.registerExtensionReader('xyz', () => activeEditors.push('second'))

    disposeFirst()
    await tabOpeners.openFile('a.xyz')

    expect(activeEditors).toStrictEqual(['second'])
  })

  it('is removed by its own disposer', async () => {
    const one = vault({ 'a.xyz': OTHER })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors: string[] = []
    const dispose = tabOpeners.registerExtensionReader('xyz', () => activeEditors.push('only'))

    dispose()
    await tabOpeners.openFile('a.xyz')

    expect(activeEditors).toStrictEqual([])
  })
})

describe('the same extension reader registered twice', () => {
  it('stays registered when the disposer of the first registration runs late', async () => {
    const one = vault({ 'a.xyz': OTHER })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors: string[] = []
    const readItem = () => activeEditors.push('read')
    const disposeFirst = tabOpeners.registerExtensionReader('xyz', readItem)
    tabOpeners.registerExtensionReader('xyz', readItem)

    disposeFirst()
    await tabOpeners.openFile('a.xyz')

    expect(activeEditors).toStrictEqual(['read'])
  })
})
