/**
 * Reaching the core the application carries.
 *
 * The native half starts it and answers with the port it came up on and the
 * token every request carries; from there everything is the schema, and both halves are generated from it.
 */
import { registerPlugin } from '@capacitor/core'
import { createBearerTransport } from '@numen/core-client'
import type { Core } from '../lib/core'
import { createFileApi } from './files'
import { createNoteApi } from './notes'
import { createVaultApi } from './vault'

interface CorePlugin {
  start(): Promise<{ port: number; dir: string; token: string }>
}

const NumenCore = registerPlugin<CorePlugin>('NumenCore')

/** Start the core and answer with what it serves on the port it came up on. */
export async function startCore(): Promise<Core> {
  const { port, dir, token } = await NumenCore.start()
  const transport = createBearerTransport(`http://127.0.0.1:${port}`, token)
  return { port, dir, ...createVaultApi(transport), ...createNoteApi(transport), ...createFileApi(transport) }
}
