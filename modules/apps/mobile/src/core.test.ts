/**
 * Where the phone reaches the core: the plugin it starts it through, and the
 * address each client asks it at.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { reach } from './core'

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

/** The addresses the clients asked, in order. */
const stubFetch = () => {
  const asked: string[] = []
  vi.stubGlobal('fetch', async (input: RequestInfo | URL) => {
    asked.push(input instanceof Request ? input.url : String(input))
    throw new Error('nothing answers')
  })
  return asked
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('reaching the core', () => {
  it('starts the core through the NumenCore plugin and reports where it is', async () => {
    const core = await reach()
    expect(core.port).toBe(PORT)
    expect(core.dir).toBe('/data/vault')
  })

  it('asks the vault service on the loopback at the port the core came up on', async () => {
    const asked = stubFetch()
    const core = await reach()
    await expect(core.vault.getVaultState({})).rejects.toThrow()
    expect(asked).toHaveLength(1)
    expect(asked[0]).toBe(`http://127.0.0.1:${PORT}/numen.v1.VaultService/GetVaultState`)
  })

  it('asks the note service on the loopback at the port the core came up on', async () => {
    const asked = stubFetch()
    const core = await reach()
    await expect(core.notes.getOpeningNote({})).rejects.toThrow()
    expect(asked).toHaveLength(1)
    expect(asked[0]).toBe(`http://127.0.0.1:${PORT}/numen.v1.NoteService/GetOpeningNote`)
  })
})
