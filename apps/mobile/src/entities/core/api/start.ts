/**
 * Reaching the core the application carries.
 *
 * The native half starts it and answers with the port it came up on and the
 * token every request carries; from there everything is the schema, and both halves are generated from it.
 */
import { registerPlugin } from '@capacitor/core'
import { createConnectTransport } from '@connectrpc/connect-web'
import type { Core } from '../lib/core'
import { createNoteApi } from './notes'
import { createVaultApi } from './vault'

interface CorePlugin {
  start(): Promise<{ port: number; dir: string; token: string }>
}

const NumenCore = registerPlugin<CorePlugin>('NumenCore')

/** Start the core and answer with what it serves on the port it came up on. */
export async function startCore(): Promise<Core> {
  const { port, dir, token } = await NumenCore.start()
  const transport = createConnectTransport({
    baseUrl: `http://127.0.0.1:${port}`,
    interceptors: [
      (next) => (req) => {
        req.header.set('Authorization', `Bearer ${token}`)
        return next(req)
      },
    ],
  })
  return { port, dir, ...createVaultApi(transport), ...createNoteApi(transport) }
}
