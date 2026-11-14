/**
 * A run whose caller awaits it, as a plugin's `context.commands.run`: it rejects with the error of
 * the run and the host reports nothing, so the caller shows the one entry. The plugins are invented.
 */
import { createMockManifest } from '@numen/core-client/testing'
import { describe, expect, it, vi } from 'vitest'
import type { CommandNote } from '@numen/plugin-sdk'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

const TIDES = 'org.example.tides'
const CREATE = 'app:file.createText'

const NOTE: CommandNote = {
  path: 'scans/mill.png',
  title: 'mill',
  type: 'note',
  isFolder: false,
  others: [],
  tab: '',
}

function createHost(hasFrontend: boolean) {
  const onError = vi.fn()
  const runBackendHandler = vi.fn(async () => {
    throw new Error('the backend is down')
  })
  const deps: PluginHostDeps = {
    getCommandDeclarations: () => [{ pluginId: TIDES, id: 'tide.read', title: 'Read the tide' }],
    getHandlerDeclarations: () => [
      { pluginId: TIDES, command: CREATE, title: 'Read the tide table', when: '', hasItems: false },
    ],
    getContentTypes: () => ({ png: 'image/png' }),
    hasPluginFrontend: () => hasFrontend,
    runBackendHandler,
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    onError,
  }
  const host = createPluginHost(deps)
  const context = host.lifecycle.createContext(
    TIDES,
    createMockManifest(TIDES, { commands: ['tide.read'], handlers: [CREATE] }),
  )
  const fail = async () => {
    throw new Error('the tide is out')
  }
  context.plugin.register.command({ id: 'tide.read', handler: fail })
  if (hasFrontend) context.plugin.register.command({ id: CREATE, handler: fail })
  return { host, onError }
}

describe('a run the caller awaits', () => {
  it('rejects with the error of a plugin command and reports nothing', async () => {
    const { host, onError } = createHost(true)

    await expect(
      host.commands.executeCommand(`${TIDES}:tide.read`, NOTE, { isAwaited: true }),
    ).rejects.toThrow('the tide is out')
    expect(onError).not.toHaveBeenCalled()
  })

  it('rejects with the error of the frontend handler of a shared command and reports nothing', async () => {
    const { host, onError } = createHost(true)

    await expect(host.commands.executeCommand(CREATE, NOTE, { isAwaited: true })).rejects.toThrow(
      'the tide is out',
    )
    expect(onError).not.toHaveBeenCalled()
  })

  it('rejects with the error of the backend handler of a shared command and reports nothing', async () => {
    const { host, onError } = createHost(false)

    await expect(host.commands.executeCommand(CREATE, NOTE, { isAwaited: true })).rejects.toThrow(
      'the backend is down',
    )
    expect(onError).not.toHaveBeenCalled()
  })
})

describe('a run without isAwaited, as from the palette', () => {
  it('reports the error of the command and resolves', async () => {
    const { host, onError } = createHost(true)

    await expect(host.commands.executeCommand(`${TIDES}:tide.read`, NOTE)).resolves.toBe(true)
    expect(onError).toHaveBeenCalledOnce()
  })
})
