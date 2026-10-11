/**
 * What a note tab does with what the window gave it, asked without a browser.
 *
 * Every operation of a tab is one the window already knows how to do for a
 * note; the tab says which note it is, and nothing else.
 */
import { describe, expect, it, vi } from 'vitest'
import { createNoteTab } from './open'
import { createNoteWikilinkExtension } from './wikilinks'
import type { createNoteChanges } from './changes'
import type { createNoteKeyboard } from './keyboard'
import type { createNoteTitles } from './titles'
import type { openNotes } from '@/entities/note'
import type { FileOpeners, WindowHandle } from '@/entities/tab'
import type { NoteTabDeps } from '../types'

vi.mock('./wikilinks', () => ({
  createNoteWikilinkExtension: vi.fn((_vault: unknown, getCurrentPath: () => string) => ({
    getCurrentPath,
  })),
}))

type Notes = ReturnType<typeof openNotes>
type Changes = ReturnType<typeof createNoteChanges>
type Keyboard = ReturnType<typeof createNoteKeyboard>
type Titles = ReturnType<typeof createNoteTitles>

/** A tab for the note `id`, standing at `Deep/Heat.md`, and everything it was asked to do. */
const tabFor = (options: { isClosing?: boolean; landed?: Map<string, string> } = {}) => {
  const isClosing = options.isClosing ?? true
  const calls: string[] = []
  const notes = {
    getPath: vi.fn((id: string) => `Deep/${id}`),
    getOpenNote: vi.fn((id: string) => ({ id, path: `Deep/${id}`, state: 'clean' })),
    getErrorMessage: vi.fn((id: string) => `error of ${id}`),
    setBody: vi.fn((id: string, body: string) => {
      calls.push(`setBody:${id}:${body}`)
    }),
    save: vi.fn((id: string) => {
      calls.push(`save:${id}`)
    }),
    keep: vi.fn((id: string) => {
      calls.push(`keep:${id}`)
    }),
    take: vi.fn((id: string) => {
      calls.push(`take:${id}`)
    }),
    close: vi.fn(async (id: string) => {
      calls.push(`close:${id}`)
      return isClosing
    }),
  }
  const changes = {
    getChange: vi.fn((path: string) => ({ path })),
    closeNote: vi.fn((path: string) => {
      calls.push(`closeNote:${path}`)
    }),
  }
  const keyboard = {
    cancelFocusRequest: vi.fn((id: string) => {
      calls.push(`cancelFocusRequest:${id}`)
    }),
    setEditor: vi.fn((id: string, editor: unknown) => {
      calls.push(`setEditor:${id}:${String(editor)}`)
    }),
    measure: vi.fn((id: string) => {
      calls.push(`measure:${id}`)
    }),
  }
  const names = {
    forgetTab: vi.fn((id: string) => {
      calls.push(`forgetTab:${id}`)
    }),
  }
  const handle = {
    closeTab: vi.fn((id: string) => {
      calls.push(`closeTab:${id}`)
    }),
  }
  const opened: [string, string, string][] = []
  const openers = {
    openFile: vi.fn(async (path: string, title: string, how: string) => {
      opened.push([path, title, how])
    }),
  }
  const resolve = vi.fn(async (from: string, written: readonly string[]) => {
    calls.push(`resolve:${from}:${written.join(',')}`)
    return options.landed ?? new Map<string, string>()
  })
  const vault = {
    resolve,
    names: async () => [],
    headings: async () => new Map(),
    create: async () => ({ ok: true as const, value: { path: '' } }),
    neighbourhood: async () => ({ focus: { title: '' } }),
  } as unknown as NoteTabDeps

  const tab = createNoteTab(
    'Heat.md',
    notes as unknown as Notes,
    changes as unknown as Changes,
    keyboard as unknown as Keyboard,
    vault,
    names as unknown as Titles,
    handle as unknown as WindowHandle,
    openers as unknown as FileOpeners,
  )
  return { tab, calls, opened, notes, changes }
}

describe('a note tab', () => {
  it('shows the note it stands for', () => {
    const { tab } = tabFor()

    expect(tab.id).toBe('Heat.md')
    expect(tab.note.value).toMatchObject({ id: 'Heat.md', path: 'Deep/Heat.md' })
  })

  it('shows the error its note ran into', () => {
    const { tab } = tabFor()

    expect(tab.errorMessage.value).toBe('error of Heat.md')
  })

  it('shows the change decided for the file its note stands at', () => {
    const { tab } = tabFor()

    expect(tab.change.value).toStrictEqual({ path: 'Deep/Heat.md' })
  })

  it('writes what is typed into its own note', () => {
    const { tab, calls } = tabFor()

    tab.updateBody('Some words')

    expect(calls).toStrictEqual(['setBody:Heat.md:Some words'])
  })

  it('saves, keeps and takes for its own note', () => {
    const { tab, calls } = tabFor()

    tab.save()
    tab.keepMine()
    tab.takeFile()

    expect(calls).toStrictEqual(['save:Heat.md', 'keep:Heat.md', 'take:Heat.md'])
  })

  it('hands its editor to the keyboard under its own note', () => {
    const { tab, calls } = tabFor()

    tab.setEditor('the editor')

    expect(calls).toStrictEqual(['setEditor:Heat.md:the editor'])
  })

  it('has the keyboard measure its own note', () => {
    const { tab, calls } = tabFor()

    tab.measure()

    expect(calls).toStrictEqual(['measure:Heat.md'])
  })
})

describe('the links a note tab completes', () => {
  it('are written from the file its note stands at', () => {
    const { tab } = tabFor()

    const [, getCurrentPath] = vi.mocked(createNoteWikilinkExtension).mock.lastCall ?? []

    expect(tab.extensions).toStrictEqual({ getCurrentPath })
    expect(getCurrentPath?.()).toBe('Deep/Heat.md')
  })
})

describe('a note tab being closed', () => {
  it('lets go of the focus owed, the change, the note, the name, and then the tab', async () => {
    const { tab, calls } = tabFor()

    tab.close('tab-1')
    await vi.waitFor(() => expect(calls).toContain('closeTab:tab-1'))

    expect(calls).toStrictEqual([
      'cancelFocusRequest:Heat.md',
      'closeNote:Deep/Heat.md',
      'close:Heat.md',
      'forgetTab:Heat.md',
      'closeTab:tab-1',
    ])
  })

  it('stays, with its name, when the note refuses to close', async () => {
    const { tab, calls } = tabFor({ isClosing: false })

    tab.close('tab-1')
    await vi.waitFor(() => expect(calls).toContain('close:Heat.md'))
    await new Promise((done) => setTimeout(done, 0))

    expect(calls).not.toContain('forgetTab:Heat.md')
    expect(calls).not.toContain('closeTab:tab-1')
  })
})

describe('a link followed from a note tab', () => {
  it('opens the note it resolves to beside the tab, under no title of its own', async () => {
    const { tab, opened } = tabFor({ landed: new Map([['name://Entropy', 'physics/Entropy.md']]) })

    tab.followLink('name://Entropy')

    await vi.waitFor(() => expect(opened).toStrictEqual([['physics/Entropy.md', '', 'beside']]))
  })

  it('asks the vault from the file of its own note', async () => {
    const { tab, calls } = tabFor()

    tab.followLink('note://Entropy')

    await vi.waitFor(() => expect(calls).toStrictEqual(['resolve:Deep/Heat.md:note://Entropy']))
  })

  it('opens nothing where the address leads to no note', async () => {
    const { tab, calls, opened } = tabFor()

    tab.followLink('name://Nowhere')
    await vi.waitFor(() => expect(calls).toHaveLength(1))
    await new Promise((done) => setTimeout(done, 0))

    expect(opened).toStrictEqual([])
  })

  it('does not ask for an address that is not a note', async () => {
    const { tab, calls } = tabFor()

    tab.followLink('https://example.com')
    await new Promise((done) => setTimeout(done, 0))

    expect(calls).toStrictEqual([])
  })
})
