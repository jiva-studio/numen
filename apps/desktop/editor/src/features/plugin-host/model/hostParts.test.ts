/** The small parts the window makes the host of its plugins from. */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { Command } from '@/entities/command'
import type { CommandTarget } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { createTestTabs } from '@/testing/tabs'
import { createAgentAsker } from './createAgentAsker'
import { createNoteKindReader } from './window/createNoteKindReader'
import { createPluginCatalog } from './createPluginCatalog'
import { createPluginReports } from './createPluginReports'
import { createWindowFacts } from './window/createWindowFacts'
import { getAppCommandIds } from './window/getAppCommandIds'

const row = (id: string, command?: string): Command => ({
  id,
  ...(command ? { command } : {}),
  text: id,
  group: 'window',
  isAvailable: () => true,
})

describe('the ids of the commands of the window', () => {
  it('list the command of a row once, whichever rows run it', () => {
    expect(
      getAppCommandIds([
        row('app:note.save'),
        row('reveal.plex', 'app:file.reveal'),
        row('reveal.files', 'app:file.reveal'),
      ]),
    ).toStrictEqual(['app:note.save', 'app:file.reveal'])
  })
})

describe('the plugins as the settings list them', () => {
  const catalog = createPluginCatalog({
    state: {
      plugins: ref([
        {
          id: 'md.example.sketch',
          name: 'Sketch',
          hasFrontend: true,
          hasBackend: false,
          readers: [{ id: 'draw', extensions: ['.sk'] }],
        },
      ]),
    },
  } as never)

  it('have a name, a frontend and a backend, and a plugin unlisted has all but a name', () => {
    expect(catalog.getName('md.example.sketch')).toBe('Sketch')
    expect(catalog.hasFrontend('md.example.sketch')).toBe(true)
    expect(catalog.hasBackend('md.example.sketch')).toBe(false)
    expect(catalog.getName('md.example.other')).toBe('md.example.other')
    expect(catalog.hasBackend('md.example.other')).toBe(true)
  })

  it('open the extensions of a reader', () => {
    expect(catalog.getReaderExtensions('md.example.sketch', 'draw')).toStrictEqual(['.sk'])
    expect(catalog.getReaderExtensions('md.example.sketch', 'none')).toStrictEqual([])
  })
})

describe('the problems of a plugin', () => {
  it('are written in the log of the plugin, under its name', () => {
    const writers = new Map<string, ReturnType<typeof vi.fn<MessageWriter>>>()
    const log = {
      getWriter: (name: string): MessageWriter => {
        const writer = writers.get(name) ?? vi.fn<MessageWriter>()
        writers.set(name, writer)
        return writer
      },
    }
    const reports = createPluginReports(log, () => 'Sketch')

    reports.reportViewIssue('md.example.sketch', 'no pane')
    reports.reportCommandIssue('md.example.sketch', 'no title')

    expect(writers.get('plugin:md.example.sketch')).toHaveBeenNthCalledWith(
      1,
      'Sketch: no pane',
      'error',
    )
    expect(writers.get('plugin:md.example.sketch')).toHaveBeenNthCalledWith(
      2,
      'Sketch: no title',
      'warning',
    )
  })
})

describe('the facts of the window the conditions read', () => {
  it('say whether a vault is open and ready, a tab is open, and a palette is open', () => {
    const target = (override: Partial<CommandTarget>) =>
      ({
        tab: 'note:1',
        vault: { id: 'sea', name: 'Sea' },
        isReady: true,
        ...override,
      }) as CommandTarget
    const read = (one: CommandTarget, isPaletteOpen: boolean) =>
      createWindowFacts({
        appCommands: [],
        getTarget: () => one,
        isPaletteOpen: () => isPaletteOpen,
      }).getValues()

    expect(read(target({}), true)).toStrictEqual({
      'window.isVaultOpen': true,
      'window.isVaultReady': true,
      'window.isTabOpen': true,
      'window.isFieldFocused': false,
      'window.isPaletteOpen': true,
    })
    expect(
      read(target({ vault: { id: '', name: '' }, tab: '', isReady: false }), false),
    ).toMatchObject({
      'window.isVaultOpen': false,
      'window.isVaultReady': false,
      'window.isTabOpen': false,
    })
  })
})

describe('the kind of the note at a path', () => {
  it('is its type and properties, and no type where the vault files none', async () => {
    const read = createNoteKindReader({
      fileKinds: (paths: readonly string[]) =>
        Promise.resolve(
          new Map(
            paths.includes('talk.md')
              ? [['talk.md', { type: 'link', properties: { media: 'a' } }]]
              : [],
          ),
        ),
    } as never)

    expect(await read('talk.md')).toStrictEqual({ noteType: 'link', properties: { media: 'a' } })
    expect(await read('none.md')).toStrictEqual({ noteType: '' })
  })
})

describe('a question put to the agent', () => {
  it('goes to the tab of the kind that takes text', async () => {
    const receiveText = vi.fn()
    const windowTabs = createTestTabs()
    windowTabs.groups.registerKinds([
      {
        kind: 'chat',
        open: () => ({ title: 'Chat' }),
        pane: {},
        getTitle: (state: { title: string }) => state.title,
        events: { receiveText },
      },
    ])
    const ask = createAgentAsker(windowTabs, vi.fn())

    ask('What is a tide?')

    await vi.waitFor(() =>
      expect(receiveText).toHaveBeenCalledWith(expect.anything(), 'What is a tide?'),
    )
  })

  it('is written as an error where no tab takes text', () => {
    const writeMessage = vi.fn()

    createAgentAsker(createTestTabs(), writeMessage)('What is a tide?')

    expect(writeMessage).toHaveBeenCalledExactlyOnceWith(
      'The agent chat is not available: its plugin is off or did not start.',
      'error',
    )
  })
})
