/**
 * Reaching the core the application carries.
 *
 * The native half starts it and answers with the port it came up on; from
 * there everything is the schema, and both halves are generated from it.
 */
import { registerPlugin } from '@capacitor/core'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { NoteService, VaultService } from '@numen/protocol'

interface Core {
  start(): Promise<{ port: number; dir: string }>
}

const NumenCore = registerPlugin<Core>('NumenCore')

/** Where the core is, and what it holds. */
export interface Reached {
  port: number
  dir: string
  /** The vault itself: what it is, and what has changed in it. */
  vault: ReturnType<typeof vaultClient>
  /** Its notes: what one holds, what it is joined to, and how one is written. */
  notes: ReturnType<typeof noteClient>
}

const reaching = (port: number) =>
  createConnectTransport({ baseUrl: `http://127.0.0.1:${port}` })

const vaultClient = (port: number) => createClient(VaultService, reaching(port))
const noteClient = (port: number) => createClient(NoteService, reaching(port))

/** Start the core and answer with clients onto the vault it opened. */
export async function reach(): Promise<Reached> {
  const { port, dir } = await NumenCore.start()
  return { port, dir, vault: vaultClient(port), notes: noteClient(port) }
}
