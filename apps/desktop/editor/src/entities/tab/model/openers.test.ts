/**
 * The single entry point that opens a vault file, tested without a window.
 *
 * A note opens in the tab kind the core chooses for it; a note no tab kind takes and
 * any other file in the reader registered for its extension, at the selector given.
 */
import { describe, expect, it, vi } from 'vitest'
import type { FileKind } from '@/shared/model/fileTypes'
import { fileOpeners, type FileOpenerDeps } from './openers'

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

const note = (type: FileKind['type'], tabKind: string = type): FileKind => ({
  isNote: true,
  type,
  tabKind,
})

const OTHER: FileKind = { isNote: false, type: 'note' }

const editorsWithNotes = (tabOpeners: ReturnType<typeof fileOpeners>) => {
  const activeEditors = editors(tabOpeners)
  tabOpeners.registerExtensionReader('md', (path, selector) =>
    activeEditors.push(`reader ${path} ${JSON.stringify(selector)}`),
  )
  return activeEditors
}

/** A vault that cannot answer at all. */
const unreachable: FileOpenerDeps = {
  fileKinds: async () => {
    throw new Error('the vault is not there')
  },
}

const editors = (tabOpeners: ReturnType<typeof fileOpeners>) => {
  const activeEditors: string[] = []
  for (const type of ['note', 'sample', 'form'] as const) {
    tabOpeners.registerEditor(type, (path, title, options) => {
      const placement = typeof options === 'string' ? options : (options?.where ?? 'current')
      const line = typeof options === 'object' ? options?.line : undefined
      activeEditors.push(`${type} ${path} ${title || '—'} ${placement} ${line ?? '—'}`)
    })
  }
  tabOpeners.registerExtensionReader('bin', (path, selector) =>
    activeEditors.push(`reader ${path} ${JSON.stringify(selector)}`),
  )
  return activeEditors
}

describe('a path opened', () => {
  it.each([
    ['a sample in the editor of its kind', 'Animals', 'sample'],
    ['a form in the editor of its kind', 'Animal', 'form'],
    ['an ordinary note in the editor of its prose', 'Entropy', 'note'],
  ] as const)('opens %s', async (_, title, type) => {
    const one = vault({ [`${title}.md`]: note(type) })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile(`${title}.md`, title)

    expect(activeEditors).toStrictEqual([`${type} ${title}.md ${title} current —`])
  })

  it('opens a note in the tab kind the core names for it, whatever its type', async () => {
    const one = vault({ 'links/tide.md': note('link', 'form') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editors(tabOpeners)

    await tabOpeners.openFile('links/tide.md', 'tide')

    expect(activeEditors).toStrictEqual(['form links/tide.md tide current —'])
  })

  it('opens a note no tab kind takes in the reader of its extension', async () => {
    const one = vault({ 'Entropy.md': note('note', '') })
    const tabOpeners = fileOpeners(one.core)
    const activeEditors = editorsWithNotes(tabOpeners)

    await tabOpeners.openFile('Entropy.md', 'Entropy')

    expect(activeEditors).toStrictEqual(['reader Entropy.md null'])
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

  it('opens in the reader of its extension where the vault could not be requested at all', async () => {
    const tabOpeners = fileOpeners(unreachable)
    const activeEditors = editorsWithNotes(tabOpeners)

    await tabOpeners.openFile('Entropy.md', 'Entropy')

    expect(activeEditors).toStrictEqual(['reader Entropy.md null'])
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

  it('opens nothing where no editor was registered for the kind', async () => {
    const one = vault({ 'Animals.md': note('sample') })
    const tabOpeners = fileOpeners(one.core)

    await expect(tabOpeners.openFile('Animals.md', 'Animals')).resolves.toBeUndefined()
  })

  it('opens a file in nothing where no reader was passed for its extension', async () => {
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
    await tabOpeners.openFileAt('Gone.md', [{ start: 1, end: 4 }])

    expect(activeEditors).toStrictEqual(['reader Physics.bin null'])
  })
})

describe('the icon of a file', () => {
  it('is the icon of the reader of its extension, and nothing where none reads it', () => {
    const tabOpeners = fileOpeners(vault().core)
    const icon = { name: 'Icon' }
    tabOpeners.registerExtensionReader('.bin', vi.fn(), icon)

    expect(tabOpeners.findFileIcon('a/b.BIN')).toBe(icon)
    expect(tabOpeners.findFileIcon('.bin')).toBeUndefined()
    expect(tabOpeners.findFileIcon('a/b.png')).toBeUndefined()
    expect(tabOpeners.findFileIcon('a.dir/bin')).toBeUndefined()
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

describe('a path opened before the plugin that opens it has started', () => {
  it('starts the plugin of the tab kind of the note, and opens the note in the kind it registers', async () => {
    const opened: string[] = []
    const tabOpeners = fileOpeners({
      ...vault({ 'Tides.md': note('note', 'tide-chart') }).core,
      startTabKind: async (kind) => {
        tabOpeners.registerEditor(kind, (path) => opened.push(`${kind} ${path}`))
      },
    })

    await tabOpeners.openFile('Tides.md', 'Tides')

    expect(opened).toStrictEqual(['tide-chart Tides.md'])
  })

  it('starts the plugin of the reader of the extension, and opens the file in the reader it registers', async () => {
    const opened: string[] = []
    const tabOpeners = fileOpeners({
      ...vault({ 'Charts/Harbor.tide': OTHER }).core,
      startExtensionReader: async (extension) => {
        tabOpeners.registerExtensionReader(extension, (path) => opened.push(`${extension} ${path}`))
      },
    })

    await tabOpeners.openFile('Charts/Harbor.tide')

    expect(opened).toStrictEqual(['tide Charts/Harbor.tide'])
  })
})
