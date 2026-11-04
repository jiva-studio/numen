/**
 * The commands the plugins declare as the window reads them, and a run as the
 * window dispatches it and reads its result.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { COMMAND_ERROR_CODE, asFailure, asValue, type PluginCatalog } from '@numen/core-client'
import { createMockCatalogEntry } from '@numen/core-client/testing'
import type { PluginInfo } from '@numen/plugin-manager'
import { useCatalogEntries } from '../model/catalogEntries'

vi.mock('@/shared/clients', () => ({ pluginCommands: request }))

const request = {
  listCommands: vi.fn(),
  runPluginCommand: vi.fn(),
}

const { canRunOn, listDeclaredCommands, runDeclaredCommand } = await import('./declaredCommands')

const convert = {
  pluginId: 'md.numen.pages',
  id: 'convert',
  title: 'Convert',
  contentTypes: ['application/pdf'],
  extensions: ['pdf'],
  arguments: [],
}

describe('the commands the plugins declare', () => {
  it('are read with their arguments in order', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'org.example.sample',
          id: 'create-sample',
          title: 'Create a sample',
          contentTypes: [],
          extensions: [],
          arguments: [
            { name: 'title', title: 'Title', isRequired: true },
            { name: 'folder', title: 'Folder', isRequired: false },
          ],
        },
      ],
    })

    const [one] = await listDeclaredCommands()

    expect(one?.pluginId).toBe('org.example.sample')
    expect(one?.arguments).toStrictEqual([
      { name: 'title', title: 'Title', isRequired: true },
      { name: 'folder', title: 'Folder', isRequired: false },
    ])
  })

  it('are read with their scope, shortcut, destination flag and the parts of their plugin', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'md.numen.tree',
          id: 'tree.revealNote',
          title: 'Show note in file tree',
          icon: 'folder-open',
          scope: 'note',
          shortcut: 'mod+shift+e',
          contentTypes: [],
          extensions: [],
          arguments: [],
        },
      ],
    })

    const [one] = await listDeclaredCommands()

    expect(one).toStrictEqual({
      pluginId: 'md.numen.tree',
      id: 'tree.revealNote',
      title: 'Show note in file tree',
      icon: 'folder-open',
      scope: 'note',
      shortcut: 'mod+shift+e',
      contentTypes: [],
      extensions: [],
      arguments: [],
    })
  })

  it('run over a file of one of their extensions, by the name of the file alone', () => {
    expect(canRunOn(convert, 'books/Bees.PDF')).toBe(true)
    expect(canRunOn(convert, '.pdf')).toBe(true)
    expect(canRunOn(convert, 'pdf')).toBe(false)
    expect(canRunOn(convert, 'books/Bees.epub')).toBe(false)
    expect(canRunOn(convert, 'books.pdf/Bees')).toBe(false)
    expect(canRunOn(convert, '')).toBe(false)
  })

  it('run over a file of any type where the core lists the extension *', () => {
    const proofread = {
      ...convert,
      id: 'proofread',
      contentTypes: ['*/*'],
      extensions: ['*'],
    }
    expect(canRunOn(proofread, 'talks/storage.wav')).toBe(true)
    expect(canRunOn(proofread, 'README')).toBe(true)
    expect(canRunOn(proofread, 'books/')).toBe(false)
  })
})

describe('a run', () => {
  it('sends the file and values and returns the plugin message', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: 'Converted 3 pages',
      openedPath: 'a.md',
    })

    const outcome = await runDeclaredCommand(convert, { path: 'a.pdf', values: { pages: '3' } })

    expect(outcome).toStrictEqual({
      ok: true,
      value: { message: 'Converted 3 pages', openedPath: 'a.md' },
    })
    expect(request.runPluginCommand).toHaveBeenLastCalledWith({
      pluginId: 'md.numen.pages',
      command: 'convert',
      path: 'a.pdf',
      arguments: { pages: '3' },
    })
  })

  it('of a command over no file is handed none', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: '',
      openedPath: '',
    })

    await runDeclaredCommand(
      { ...convert, contentTypes: [], extensions: [] },
      { path: 'a.md', values: {} },
    )

    expect(request.runPluginCommand).toHaveBeenLastCalledWith(expect.objectContaining({ path: '' }))
  })

  it('that did not succeed answers why', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.FAILED,
      message: 'the disk is full',
    })

    const outcome = await runDeclaredCommand(convert, { path: 'a.pdf', values: {} })

    expect(outcome).toStrictEqual({ ok: false, error: 'the disk is full' })
  })
})

describe('useCatalogEntries', () => {
  const createPlugin = (id: string, isActive: boolean): PluginInfo => ({
    id,
    name: id,
    version: '1.0.0',
    description: '',
    isActive,
    capabilities: [],
    secrets: [],
    storedSecrets: [],
    hasFrontend: false,
    hasBackend: false,
    settings: [],
  })

  it('starts empty and refreshes only when plugin ids or active states change', async () => {
    const entry = createMockCatalogEntry('sample.notes')
    const list = vi.fn<PluginCatalog['list']>().mockResolvedValue(
      asValue({
        entries: [entry],
        sourceErrors: [{ source: 'community', message: 'timed out' }] as never,
      }),
    )
    const catalog = {
      list,
      findByTabKind: vi.fn(),
    } as unknown as PluginCatalog
    const plugins = ref<readonly PluginInfo[]>([])
    const errors: unknown[] = []
    const entries = useCatalogEntries(catalog, plugins, (err) => errors.push(err))

    expect(entries.value).toStrictEqual([])
    expect(list).not.toHaveBeenCalled()

    plugins.value = [createPlugin('a', true), createPlugin('b', false)]
    await nextTick()
    await flushPromises()
    expect(list).toHaveBeenCalledTimes(1)
    expect(entries.value).toStrictEqual([entry])
    expect(errors).toHaveLength(1)
    expect((errors[0] as Error).message).toBe('The plugin source community failed: timed out')

    // Same ids and active states do not trigger a refresh
    plugins.value = [createPlugin('a', true), createPlugin('b', false)]
    await nextTick()
    await flushPromises()
    expect(list).toHaveBeenCalledTimes(1)

    // Boundary between plugins matters (joining with newline vs empty string)
    plugins.value = [createPlugin('a:trueb', false)]
    await nextTick()
    await flushPromises()
    expect(list).toHaveBeenCalledTimes(2)

    // Unavailable error reports reason, and other error kinds report default fallback
    list.mockResolvedValueOnce(asFailure({ kind: 'unavailable', message: 'offline' }))
    plugins.value = [createPlugin('a', false)]
    await nextTick()
    await flushPromises()
    expect((errors.at(-1) as Error).message).toBe('The plugin catalog is unavailable: offline')

    list.mockResolvedValueOnce(asFailure({ kind: 'other' as 'unavailable', message: 'offline' }))
    plugins.value = [createPlugin('a', true)]
    await nextTick()
    await flushPromises()
    expect((errors.at(-1) as Error).message).toBe(
      'The plugin catalog is unavailable: it lists nothing',
    )
  })
})
