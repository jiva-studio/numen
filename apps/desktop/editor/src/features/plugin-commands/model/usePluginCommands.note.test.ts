/**
 * A plugin's command whose condition reads the type or the properties of the note: the palette reads
 * the note in front from the core and lists the command where the condition holds over it.
 */
import { describe, expect, it } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import type { PluginHost } from '@numen/plugin-loader'
import { createCommandRegistry } from '@/shared/model/command'
import type { CommandTarget, RunSupport } from '@/shared/model/command'
import type { FileKind } from '@/shared/model/fileTypes'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'

const RUNS: RunSupport = { canRun: () => true, cannotRun: () => undefined }

const KINDS: Readonly<Record<string, FileKind>> = {
  'links/mill.md': { isNote: true, type: 'link', properties: { media: '' } },
  'notes/pond.md': { isNote: true, type: 'note' },
}

const createHost = () =>
  ({
    commands: {
      hasCommandItems: () => false,
      getPaletteCommands: () => [
        {
          id: 'md.media:link.downloadMedia',
          title: 'Download media',
          when: "target.noteType == 'link'",
        },
      ],
    },
    keybindings: { getKeyFor: () => undefined },
  }) as unknown as PluginHost

const createEntry = () => {
  const reads: string[] = []
  const commands = usePluginCommands(
    createCommandRegistry(),
    createCommandNoteReader({
      fileKinds: async (paths) => {
        reads.push(...paths)
        return new Map(paths.flatMap((one) => (KINDS[one] ? [[one, KINDS[one]] as const] : [])))
      },
    }),
    createHost(),
  )
  const [entry] = commands.getPaletteEntries()
  return { entry, reads, commands }
}

const createTarget = (path: string) =>
  ({ tab: 't1', path, file: path, title: '', isReady: true, vault: { id: 'v1' } }) as CommandTarget

describe('a command whose condition reads the type of the note', () => {
  it('is available over a note of that type once the note is read', async () => {
    const { entry } = createEntry()
    const target = createTarget('links/mill.md')

    expect(entry?.isAvailable(target, RUNS)).toBe(false)
    await flushPromises()

    expect(entry?.isAvailable(target, RUNS)).toBe(true)
  })

  it('is not available over a note of another type', async () => {
    const { entry } = createEntry()
    const target = createTarget('notes/pond.md')

    entry?.isAvailable(target, RUNS)
    await flushPromises()

    expect(entry?.isAvailable(target, RUNS)).toBe(false)
  })

  it('reads the note once, and again after the palette notes are forgotten', async () => {
    const { entry, reads, commands } = createEntry()
    const target = createTarget('links/mill.md')

    entry?.isAvailable(target, RUNS)
    entry?.isAvailable(target, RUNS)
    await flushPromises()
    commands.forgetPaletteNotes()
    entry?.isAvailable(target, RUNS)
    await flushPromises()

    expect(reads).toStrictEqual(['links/mill.md', 'links/mill.md'])
  })
})
