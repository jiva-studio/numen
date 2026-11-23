import { describe, expect, it, vi } from 'vitest'
import { createNoteAnswers } from '@/testing/noteCore'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { NOTE_ERROR_CODE } from '@numen/core-client'
import { asValue } from '@numen/result'
import { createTestTabs } from '@/testing/tabs'
import { createMessageLog } from '@/shared/model/notices/messages'
import type { AnyTabKind } from '@numen/plugin-loader'
import { createNoteEditors } from './createNoteEditors'

const settle = async () => {
  for (let i = 0; i < 10; i++) await new Promise((resolve) => setTimeout(resolve, 0))
}

const create = (windowTabs = createTestTabs()) => {
  const api = createAnsweredApi(
    createNoteAnswers({
      read: async () => asValue({ body: 'High tide at six.', at: '1:0: 1 at:High tide at six.' }),
      write: async () => asValue({ body: '', at: '1:0: 1 at:' }),
    }),
  )
  return createNoteEditors({
    api: {
      notes: api.vault.note,
      noteMaking: { createNote: api.vault.note.createNote, relations: api.vault.relations },
      quit: api.window.quit,
    },
    log: createMessageLog(),
    windowTabs,
  })
}

describe('the editors of a window', () => {
  it('hold no note before one is opened', () => {
    const editors = create()

    expect(editors.store.has('note:1')).toBe(false)
    expect(editors.store.getTabAt('Tides.md')).toBeNull()
    expect(editors.getTitle('note:1')).toBe('note:1')
  })

  it('hold an open note by its tab, and name it by its path', () => {
    const editors = create()

    editors.notes.open('note:1', 'Tides.md')

    expect(editors.store.has('note:1')).toBe(true)
    expect(editors.store.getPath('note:1')).toBe('Tides.md')
    expect(editors.store.getTabAt('Tides.md')).toBe('note:1')
    expect(editors.store.getTabAt('Moon.md')).toBeNull()
    expect(editors.getTitle('note:1')).toBe('Tides.md')
    expect(editors.store.isAsking('note:1')).toBe(false)
  })

  it('name a note by the title of the tab it is open in', async () => {
    const windowTabs = createTestTabs()
    windowTabs.kinds.registerKinds([
      {
        kind: 'note',
        open: (path: string) => ({ title: path === 'Tides.md' ? 'Tide table' : 'Other tab' }),
        pane: {},
        getTitle: (state: { title: string }) => state.title,
      } as AnyTabKind,
    ])
    const editors = create(windowTabs)
    await windowTabs.tabs.openTabOfKind('note', 'Other.md')
    const id = await windowTabs.tabs.openTabOfKind('note', 'Tides.md')

    editors.notes.open(id, 'Tides.md')

    expect(editors.getTitle(id)).toBe('Tide table')
  })

  it('flushes a note, answers a quit, and logs a note that cannot be made', async () => {
    let diskBody = 'High tide at six.'
    const signals: AbortSignal[] = []
    const reported: string[] = []
    const api = createAnsweredApi({
      ...createNoteAnswers({
        read: async () => asValue({ body: diskBody, at: `1:0: 1 at:${diskBody}` }),
        write: async (_path, body) => {
          diskBody = body
          return asValue({ body: '', at: `1:0: 1 at:${body}` })
        },
      }),
      notes: {
        ...createNoteAnswers({
          read: async () => asValue({ body: diskBody, at: `1:0: 1 at:${diskBody}` }),
          write: async (_path, body) => {
            diskBody = body
            return asValue({ body: '', at: `1:0: 1 at:${body}` })
          },
        }).notes,
        createNote: async () => ({ error: NOTE_ERROR_CODE.INVALID_NAME }),
      },
      windowService: {
        watchWindowClose: async function* (_req, context) {
          signals.push(context.signal)
          yield { token: 'quit-1', isFlushRequired: true }
          await new Promise((resolve) => context.signal.addEventListener('abort', resolve))
        },
        reportPendingWork: async ({ token, result }) => {
          reported.push(`${token}:${result}`)
          return {}
        },
      },
    })
    const log = createMessageLog()
    const editors = createNoteEditors({
      api: {
        notes: api.vault.note,
        noteMaking: { createNote: api.vault.note.createNote, relations: api.vault.relations },
        quit: api.window.quit,
      },
      log,
      windowTabs: createTestTabs(),
    })

    await editors.making.createWithTitle('..', '', null)
    expect(log.messages.value.some((m) => m.severity === 'error')).toBe(true)

    editors.notes.open('note:1', 'Tides.md')
    await settle()
    editors.notes.setBody('note:1', 'Low tide at noon.')
    await editors.store.flushPending('note:1')
    expect(diskBody).toBe('Low tide at noon.')

    const quitPromise = editors.fileFlush.start()
    await vi.waitFor(() => expect(reported.length).toBeGreaterThan(0))
    editors.close()
    await quitPromise
    expect(signals[0]?.aborted).toBe(true)
  })
})
