import { describe, expect, it } from 'vitest'
import { effectScope } from 'vue'
import { openNotes as openNotesOn } from '@/entities/note'
import { createCustomTabKind, type PluginHostDeps } from '@numen/plugin-loader'
import { blank } from '@/testing/stubs'
import { answerNotes, type NoteCore as Notes } from '@/testing/noteCore'

const statFile = (body: string): string => `1 1 at:${body}`

const openNotes = (core: Notes, how: Parameters<typeof openNotesOn>[0]) => {
  answerNotes(core)
  return openNotesOn(how)
}

function createFiles() {
  const files = new Map<string, string>()
  const core: Notes = {
    read: async (path) =>
      files.has(path)
        ? { ok: true, value: { body: files.get(path) ?? '', at: statFile(files.get(path) ?? '') } }
        : { ok: false, error: 'notFound' },
    write: async (path, body, seen) => {
      const existing = files.get(path)
      if (
        seen &&
        existing !== undefined &&
        existing !== seen.prose &&
        statFile(existing) !== seen.stat
      ) {
        return { ok: false, error: 'changed' }
      }
      files.set(path, body)
      return { ok: true, value: { body: '', at: statFile(body) } }
    },
  }
  return { core, files }
}

const wait = (ms = 20) => new Promise((resolve) => setTimeout(resolve, ms))

interface BufferedState {
  readonly path: string
  readonly title: string
  readonly body: { readonly value: string }
  readonly fileConflict: { readonly value: string | null }
  updateBody(body: string): void
  keepMine(): void
  takeFile(): void
}

function openNoteTab(path: string) {
  const { core, files } = createFiles()
  files.set(path, 'The tide turns at noon.')
  const notes = openNotes(core, { limits: { quiet: 1, bound: 5 } })
  const kind = createCustomTabKind(
    'md.numen.notes',
    { kind: 'note', component: blank, title: 'Note' },
    { getNoteBuffer: () => notes } as unknown as PluginHostDeps,
  )
  const scope = effectScope(true)
  const state = scope.run(() => kind.open(path, 'note:tab')) as unknown as BufferedState
  return { state, notes, files, scope }
}

describe('a note tab of the note kind', () => {
  it('shows the note read from the store and writes what is typed once typing settles', async () => {
    const { state, files } = openNoteTab('notes/Tide.md')
    await wait()
    expect(state.body.value).toBe('The tide turns at noon.')

    state.updateBody('The tide turns at dusk.')
    await wait()

    expect(files.get('notes/Tide.md')).toBe('The tide turns at dusk.')
  })

  it('shows a change made on disk at once when nothing is unsaved', async () => {
    const { state, notes, files } = openNoteTab('notes/Tide.md')
    await wait()

    files.set('notes/Tide.md', 'Written by another program.')
    notes.applyPathChanges(['notes/Tide.md'])
    await wait()

    expect(state.body.value).toBe('Written by another program.')
    expect(state.fileConflict.value).toBeNull()
  })

  it('pauses saving when the file changed under unsaved text, and keeps the text on request', async () => {
    const { state, files } = openNoteTab('notes/Tide.md')
    await wait()

    files.set('notes/Tide.md', 'Written by another program.')
    state.updateBody('Typed in the window.')
    await wait()

    expect(state.fileConflict.value).toBe('stale')
    expect(files.get('notes/Tide.md')).toBe('Written by another program.')

    state.keepMine()
    await wait()

    expect(files.get('notes/Tide.md')).toBe('Typed in the window.')
    expect(state.fileConflict.value).toBeNull()
  })

  it('takes the file and drops the unsaved text on request', async () => {
    const { state, files } = openNoteTab('notes/Tide.md')
    await wait()
    files.set('notes/Tide.md', 'Written by another program.')
    state.updateBody('Typed in the window.')
    await wait()

    state.takeFile()
    await wait()

    expect(state.body.value).toBe('Written by another program.')
    expect(state.fileConflict.value).toBeNull()
    expect(files.get('notes/Tide.md')).toBe('Written by another program.')
  })

  it('follows its note to the name it is renamed to', async () => {
    const { state, notes, files } = openNoteTab('notes/Tide.md')
    await wait()

    files.set('notes/Ebb.md', files.get('notes/Tide.md') ?? '')
    files.delete('notes/Tide.md')
    notes.applyPathChanges(
      ['notes/Tide.md', 'notes/Ebb.md'],
      [{ from: 'notes/Tide.md', to: 'notes/Ebb.md' }],
    )
    await wait()

    expect(state.path).toBe('notes/Ebb.md')
    expect(state.title).toBe('Ebb')
    state.updateBody('Low water.')
    await wait()
    expect(files.get('notes/Ebb.md')).toBe('Low water.')
    expect(files.has('notes/Tide.md')).toBe(false)
  })

  it('writes what is unsaved and lets the note go when the tab closes', async () => {
    const { state, notes, files, scope } = openNoteTab('notes/Tide.md')
    await wait()

    state.updateBody('Typed before closing.')
    scope.stop()
    await wait()

    expect(files.get('notes/Tide.md')).toBe('Typed before closing.')
    expect(notes.has('note:tab')).toBe(false)
  })
})

describe('the links of a note tab of the note kind', () => {
  /** A tab whose host answers searches, headings and addresses from what a test puts in it. */
  function openLinkingTab() {
    const openCalls: { path: string; placement?: string }[] = []
    const { core, files } = createFiles()
    files.set('notes/Tide.md', 'See [[Entropy]].')
    const notes = openNotes(core, { limits: { quiet: 1, bound: 5 } })
    const kind = createCustomTabKind(
      'md.numen.notes',
      { kind: 'note', component: blank, title: 'Note' },
      {
        getNoteBuffer: () => notes,
        getNotes: () => ({
          findNotes: async (query: string) =>
            query.toLowerCase().startsWith('en')
              ? [{ path: 'notes/Entropy.md', title: 'Entropy' }]
              : [],
          getHeadings: async () => new Map([['notes/Entropy.md', [{ text: 'Shannon formula' }]]]),
        }),
        getWorkspace: () => ({
          files: {
            open: (path: string, options?: { placement?: string }) =>
              openCalls.push({
                path,
                ...(options?.placement ? { placement: options.placement } : {}),
              }),
          },
        }),
        resolveLink: async (_from: string, address: string) =>
          address === 'name://Entropy' ? 'notes/Entropy.md' : '',
      } as unknown as PluginHostDeps,
    )
    const scope = effectScope(true)
    const state = scope.run(() => kind.open('notes/Tide.md', 'note:tide')) as unknown as {
      followLink(address: string): void
      wikilinks: {
        search(query: string): Promise<{ title: string }[]>
        headings(title: string): Promise<string[]>
      }
    }
    return { state, openCalls }
  }

  it('opens the note an address points to next to the tab', async () => {
    const { state, openCalls } = openLinkingTab()

    state.followLink('name://Entropy')
    await wait()

    expect(openCalls).toStrictEqual([{ path: 'notes/Entropy.md', placement: 'split' }])
  })

  it('opens nothing for an address that lands on no note', async () => {
    const { state, openCalls } = openLinkingTab()

    state.followLink('name://Nowhere')
    await wait()

    expect(openCalls).toStrictEqual([])
  })

  it('offers the notes of the vault by name and the headings of a named note', async () => {
    const { state } = openLinkingTab()

    expect(await state.wikilinks.search('Entr')).toStrictEqual([{ title: 'Entropy', type: 'note' }])
    expect(await state.wikilinks.headings('Entropy')).toStrictEqual(['Shannon formula'])
  })
})

describe('note tab path after the store closes its note', () => {
  it.each(['dd.md', 'Books/x.md', 'notes/Мысли о приливе.md', 'notes/two words.md'])(
    'keeps the path %s that the open tabs save',
    async (path) => {
      const { state, notes } = openNoteTab(path)
      await wait()

      await notes.flush()
      await wait()

      expect(state.path).toBe(path)
      expect(state.title).toBe(path.split('/').pop()?.replace(/\.md$/, ''))
    },
  )
})

describe('note tab restored from the saved path', () => {
  it.each(['dd.md', 'Books/x.md', 'notes/Мысли о приливе.md', 'notes/two words.md'])(
    'reads %s again after the store let go of it, in two restarts',
    async (path) => {
      const first = openNoteTab(path)
      await wait()
      await first.notes.flush()
      await wait()
      const saved = (
        createCustomTabKind('md.numen.notes', {
          kind: 'note',
          component: blank,
          title: 'Note',
        }).address?.getOpenTab?.(first.state) ?? { path: '' }
      ).path
      expect(saved).toBe(path)

      const second = openNoteTab(saved)
      await wait()

      expect(second.state.body.value).toBe('The tide turns at noon.')
      expect(second.state.fileConflict.value).toBeNull()
    },
  )
})
