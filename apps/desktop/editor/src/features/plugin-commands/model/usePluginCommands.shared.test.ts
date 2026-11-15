/**
 * The shared commands in the palette: one entry for each the enabled plugins handle, whose pick
 * list is its handlers, run over the target with the one picked.
 */
import { createCommandRegistry } from '@/shared/model/command'
import { describe, expect, it, vi } from 'vitest'
import type {
  SHARED_COMMANDS,
  HandlerDeclaration,
  PluginHost,
} from '@numen/plugin-loader'
import type { CommandTarget, RunSupport } from '@/shared/model/command'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createSharedCommands } from '../lib/sharedCommands'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'

type SharedCommand = (typeof SHARED_COMMANDS)[number]

/** The kinds of files, over a core that answers none. */
const { getFileKinds } = createAnsweredApi({}).vault.file

const HANDLER: HandlerDeclaration = {
  pluginId: 'md.pages',
  command: 'app:file.createText',
  title: 'Extract text layer',
  hasItems: false,
}

const RUNS: RunSupport = { canRun: () => true, cannotRun: () => undefined }

const createHost = () =>
  ({
    commands: {
      hasCommandItems: () => true,
      getPaletteCommands: () => [],
      executeCommand: vi.fn(async () => true),
    },
  }) as unknown as PluginHost

describe('a shared command in the palette', () => {
  it('is an entry with a pick list while a plugin handles it, and none otherwise', () => {
    const handled = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      createHost(),
      { getHandlers: () => [HANDLER] },
    )
    const unhandled = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      createHost(),
      { getHandlers: () => [] },
    )

    expect(handled.getPaletteEntries()).toMatchObject([
      { id: 'app:file.createText', text: 'Create text', input: 'items', group: 'note' },
    ])
    expect(unhandled.getPaletteEntries()).toStrictEqual([])
  })

  it('is available over a file and not over the window alone', () => {
    const [entry] = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      createHost(),
      { getHandlers: () => [HANDLER] },
    ).getPaletteEntries()
    const target = { path: 'scan.pdf', file: 'scan.pdf', isReady: true } as CommandTarget

    expect(entry?.isAvailable(target, RUNS)).toBe(true)
    expect(entry?.isAvailable({ ...target, path: '' }, RUNS)).toBe(true)
    expect(entry?.isAvailable({ ...target, file: '' }, RUNS)).toBe(true)
    expect(entry?.isAvailable({ ...target, path: '', file: '' }, RUNS)).toBe(false)
  })

  it('prompts for required arguments and checks handler availability when a shared command declares arguments', () => {
    const sharedWithArgs: readonly SharedCommand[] = [
      {
        id: 'app:note.import',
        title: 'Import note',
        arguments: [
          { name: 'url', title: 'URL', type: 'string', isRequired: true },
          { name: 'tag', title: 'Tag', type: 'string', isRequired: false },
        ],
      },
      { id: 'app:file.createText', title: 'Create text', arguments: [] },
    ]
    const emptyShared = createSharedCommands(() => undefined, () => true, sharedWithArgs)
    expect(emptyShared.getSharedCommands()).toStrictEqual([])
    expect(emptyShared.getSharedEntries()).toStrictEqual([])

    const handled = createSharedCommands(
      () => [
        {
          pluginId: 'md.importer',
          command: 'app:note.import',
          title: 'Import URL',
          when: 'target.isNote',
          hasItems: false,
        },
      ],
      (when, one) => when === 'target.isNote' && one.path.endsWith('.md'),
      sharedWithArgs,
    )
    expect(handled.getSharedCommands()).toStrictEqual([
      { id: 'app:note.import', title: 'Import note' },
    ])
    const [entry] = handled.getSharedEntries()
    expect(entry).toMatchObject({
      id: 'app:note.import',
      text: 'Import note',
      group: 'window',
      input: 'arguments',
      prompts: [{ name: 'url', title: 'URL', isRequired: true }],
    })
    const noteTarget = { path: 'a.md', file: 'a.md', isReady: true } as CommandTarget
    expect(entry?.isAvailable(noteTarget, RUNS)).toBe(true)
    expect(entry?.isAvailable({ ...noteTarget, isReady: false }, RUNS)).toBe(false)
    expect(entry?.isAvailable({ ...noteTarget, path: 'a.pdf' }, RUNS)).toBe(false)
  })
})
