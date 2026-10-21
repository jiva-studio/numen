/**
 * Where the phone reaches the core: the plugin it starts it through, and the
 * address each client asks it at.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { startCore } from './core'

const PORT = 41234

const plugin = vi.hoisted(() => ({
  start: vi.fn(async () => ({ port: 41234, dir: '/data/vault' })),
}))

vi.mock('@capacitor/core', () => ({
  registerPlugin: (name: string) =>
    name === 'NumenCore'
      ? plugin
      : { start: async () => Promise.reject(new Error(`no plugin named ${name}`)) },
}))

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('reaching the core', () => {
  it('starts the core through the NumenCore plugin and asks each service on the loopback at its port', async () => {
    const request: string[] = []
    vi.stubGlobal('fetch', async (input: RequestInfo | URL) => {
      request.push(input instanceof Request ? input.url : String(input))
      throw new Error('nothing answers')
    })

    const core = await startCore()
    expect(core.port).toBe(PORT)
    expect(core.dir).toBe('/data/vault')

    await expect(core.vault.getVaultState({})).rejects.toThrow()
    await expect(core.notes.getOpeningNote({})).rejects.toThrow()
    expect(request).toEqual([
      `http://127.0.0.1:${PORT}/numen.v1.VaultService/GetVaultState`,
      `http://127.0.0.1:${PORT}/numen.v1.NoteService/GetOpeningNote`,
    ])
  })
})
