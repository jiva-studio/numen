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
  listCommandItems: vi.fn(),
}

const { isOverTarget, listDeclaredCommandItems, listDeclaredCommands, runDeclaredCommand } =
  await import('./declaredCommands')

const proofread = {
  pluginId: 'org.example.ink',
  id: 'ink.proofread',
  title: 'Proofread text',
  when: 'version.isLive',
  hasItems: true as const,
  arguments: [],
}

const convert = {
  pluginId: 'md.numen.pages',
  id: 'convert',
  title: 'Convert',
  when: "target.contentType == 'application/pdf'",
  arguments: [],
}

describe('the slot entries the plugins declare', () => {
  it('are read with what each places, its look, its group, its order and its condition', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      slots: [
        {
          pluginId: 'org.example.ledger',
          entry: {
            slot: 'fileTree.context',
            placed: { case: 'submenu', value: 'ledger.new' },
            title: 'New ledger',
            icon: 'plus',
            group: '0-new',
            order: 0,
            when: 'isFolder && !isMultiple',
          },
        },
        {
          pluginId: 'org.example.ledger',
          entry: {
            slot: 'titleBar.status',
            placed: { case: 'view', value: 'balance' },
            title: '',
            icon: '',
            group: '',
            when: '',
          },
        },
      ],
    })

    const { slots } = await listDeclaredCommands()

    expect(slots).toStrictEqual([
      {
        pluginId: 'org.example.ledger',
        slot: 'fileTree.context',
        submenu: 'ledger.new',
        title: 'New ledger',
        iconName: 'plus',
        group: '0-new',
        order: 0,
        when: 'isFolder && !isMultiple',
      },
      { pluginId: 'org.example.ledger', slot: 'titleBar.status', view: 'balance' },
    ])
  })
})

describe('the commands the plugins declare', () => {
  it('are read with their arguments in order', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'org.example.sample',
          id: 'create-sample',
          title: 'Create a sample',
          arguments: [
            { name: 'title', title: 'Title', isRequired: true },
            { name: 'folder', title: 'Folder', isRequired: false },
          ],
        },
      ],
      slots: [],
    })

    const {
      commands: [one],
    } = await listDeclaredCommands()

    expect(one?.pluginId).toBe('org.example.sample')
    expect(one?.arguments).toStrictEqual([
      { name: 'title', title: 'Title', isRequired: true },
      { name: 'folder', title: 'Folder', isRequired: false },
    ])
  })

  it('are read with their condition, shortcut, destination flag and the parts of their plugin', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'md.numen.tree',
          id: 'tree.revealNote',
          title: 'Show note in file tree',
          icon: 'folder-open',
          when: 'target.isNote',
          shortcut: 'mod+shift+e',
          arguments: [],
        },
      ],
      slots: [],
    })

    const {
      commands: [one],
    } = await listDeclaredCommands()

    expect(one).toStrictEqual({
      pluginId: 'md.numen.tree',
      id: 'tree.revealNote',
      title: 'Show note in file tree',
      icon: 'folder-open',
      when: 'target.isNote',
      shortcut: 'mod+shift+e',
      arguments: [],
    })
  })

  it('are read as offering items where the core marks them', async () => {
    request.listCommands.mockResolvedValue({
      commands: [{ ...proofread, hasItems: true }],
      slots: [],
    })

    const {
      commands: [one],
    } = await listDeclaredCommands()

    expect(one).toStrictEqual(proofread)
  })

  it('run over a target where their condition reads one', () => {
    expect(isOverTarget(convert)).toBe(true)
    expect(isOverTarget({ when: 'window.isVaultReady' })).toBe(false)
    expect(isOverTarget({})).toBe(false)
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
      { ...convert, when: 'window.isVaultReady' },
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

describe('the items of a command', () => {
  it('are the items the core lists over the file', async () => {
    request.listCommandItems.mockResolvedValue({
      items: [{ id: 'org.example.ears/ears', title: 'transcript (live)' }],
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: '',
    })

    expect(await listDeclaredCommandItems(proofread, 'talks/a.mp3')).toStrictEqual([
      { id: 'org.example.ears/ears', title: 'transcript (live)' },
    ])
    expect(request.listCommandItems).toHaveBeenLastCalledWith({
      pluginId: 'org.example.ink',
      command: 'ink.proofread',
      path: 'talks/a.mp3',
    })
  })

  it('that cannot be listed fail with why', async () => {
    request.listCommandItems.mockResolvedValue({
      items: [],
      error: COMMAND_ERROR_CODE.FAILED,
      message: 'talks/a.mp3 is not in the vault',
    })

    await expect(listDeclaredCommandItems(proofread, 'talks/a.mp3')).rejects.toThrow(
      'talks/a.mp3 is not in the vault',
    )
  })

  it('run the command over the file with the item picked as its argument', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: '',
      openedPath: '',
    })

    await runDeclaredCommand(proofread, {
      path: 'talks/a.mp3',
      values: {},
      argument: 'org.example.ears/ears',
    })

    expect(request.runPluginCommand).toHaveBeenLastCalledWith({
      pluginId: 'org.example.ink',
      command: 'ink.proofread',
      path: 'talks/a.mp3',
      arguments: {},
      argument: 'org.example.ears/ears',
    })
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
