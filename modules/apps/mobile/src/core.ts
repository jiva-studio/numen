/**
 * Reaching the core the application carries.
 *
 * The native half starts it and answers with the port it came up on; from
 * there everything is the schema, and both halves are generated from it.
 */
import { registerPlugin } from '@capacitor/core'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { VaultService } from '@numen/protocol'

interface Core {
  start(): Promise<{ port: number; dir: string }>
}

const NumenCore = registerPlugin<Core>('NumenCore')

/** Where the core is, and what it holds. */
export interface Reached {
  port: number
  dir: string
  vault: ReturnType<typeof client>
}

const client = (port: number) =>
  createClient(VaultService, createConnectTransport({ baseUrl: `http://127.0.0.1:${port}` }))

/** Start the core and answer with a client onto the vault it opened. */
export async function reach(): Promise<Reached> {
  const { port, dir } = await NumenCore.start()
  return { port, dir, vault: client(port) }
}
