/**
 * The services a scenario names: programs of the test that stand in for a language model server on
 * this computer, each with a name, an address and the way it answers. A service no step sets up
 * works and takes any key; it starts when a connection first needs its address. A connection keeps
 * the address it was given, so a service set up again listens where it listened before. The program is the fake
 * OpenAI server of the plugin "Language model services": the CI jobs unpack it, and
 * `make app_desktop_acceptance` builds it.
 */
import { spawn, type ChildProcess } from 'node:child_process'
import { once } from 'node:events'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { freeFromRemoval, holdForRemoval } from './reaper.js'

const PLUGIN = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../plugins/connections-llm')
const BINARY = path.join(PLUGIN, 'bin', 'fakeopenai')
const BUSY_STATUS = 429
/** A busy service asks to wait this long, so it stays busy for the rest of a scenario. */
const RETRY_AFTER_SECONDS = 600

interface ServiceState {
  isBusy: boolean
  key: string
}

interface RunningService {
  readonly child: ChildProcess
  readonly address: string
}

const requireBinary = (): string => {
  if (!fs.existsSync(BINARY)) {
    throw new Error(`${BINARY} is missing; build it with make app_desktop_acceptance`)
  }
  return BINARY
}

export const createServiceFlags = (state: ServiceState): string[] => [
  ...(state.isBusy ? ['-status', String(BUSY_STATUS), '-retry-after', String(RETRY_AFTER_SECONDS)] : []),
  ...(state.key === '' ? [] : ['-key', state.key]),
]

const start = (listen: string, flags: readonly string[]): Promise<RunningService> =>
  new Promise((resolve, reject) => {
    const child = spawn(requireBinary(), ['-addr', listen, ...flags], { stdio: ['ignore', 'pipe', 'inherit'] })
    if (child.pid !== undefined) holdForRemoval('pid', child.pid)
    child.once('error', reject)
    child.once('exit', () => reject(new Error('the service ended before it listened')))
    child.stdout.once('data', (chunk: Buffer) => {
      child.removeAllListeners('exit')
      resolve({ child, address: chunk.toString().trim() })
    })
  })

export class ServiceDriver {
  private readonly states = new Map<string, ServiceState>()
  private readonly running = new Map<string, RunningService>()

  /** Makes the service refuse every request and ask to wait. */
  async setBusy(name: string): Promise<void> {
    await this.restart(name, { ...this.getState(name), isBusy: true })
  }

  /** Makes the service refuse a request that does not carry the key. */
  async setKey(name: string, key: string): Promise<void> {
    await this.restart(name, { ...this.getState(name), key })
  }

  async getAddress(name: string): Promise<string> {
    const service = this.running.get(name)
    if (service !== undefined) return service.address
    await this.restart(name, this.getState(name))
    return this.getAddress(name)
  }

  stopAll(): void {
    for (const name of [...this.running.keys()]) this.stop(name)
  }

  private getState(name: string): ServiceState {
    return this.states.get(name) ?? { isBusy: false, key: '' }
  }

  private async restart(name: string, state: ServiceState): Promise<void> {
    const previous = this.running.get(name)
    this.stop(name)
    if (previous !== undefined && previous.child.exitCode === null && previous.child.signalCode === null) {
      await once(previous.child, 'exit')
    }
    this.states.set(name, state)
    const listen = previous === undefined ? '127.0.0.1:0' : new URL(previous.address).host
    this.running.set(name, await start(listen, createServiceFlags(state)))
  }

  private stop(name: string): void {
    const service = this.running.get(name)
    this.running.delete(name)
    if (service === undefined) return
    service.child.kill()
    if (service.child.pid !== undefined) freeFromRemoval('pid', service.child.pid)
  }
}
