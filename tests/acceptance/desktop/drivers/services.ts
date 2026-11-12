/**
 * The services a scenario names: programs of the test that stand in for a language model server on
 * this computer, each with a name, an address and the way it answers. The program is the fake
 * OpenAI server of the plugin "Language model services": the CI jobs unpack it, and a local run
 * builds it on the first use.
 */
import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { freeFromRemoval, holdForRemoval } from './reaper.js'

const PLUGIN = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../plugins/connections-llm')
const BINARY = path.join(PLUGIN, 'bin', 'fakeopenai')
const WORKING = 200
/** A service that refuses asks to wait this long, so it stays refused for the rest of a scenario. */
const RETRY_AFTER_SECONDS = 600

interface ServiceState {
  status: number
  key: string
}

interface RunningService {
  readonly child: ChildProcess
  readonly address: string
}

const requireBinary = (): string => {
  if (fs.existsSync(BINARY)) return BINARY
  const partial = `${BINARY}.${process.pid}`
  fs.mkdirSync(path.dirname(BINARY), { recursive: true })
  execFileSync('go', ['build', '-o', partial, './backend/cmd/fakeopenai'], { cwd: PLUGIN, stdio: 'inherit' })
  fs.renameSync(partial, BINARY)
  return BINARY
}

export const createServiceFlags = (state: ServiceState): string[] => [
  ...(state.status === WORKING ? [] : ['-status', String(state.status), '-retry-after', String(RETRY_AFTER_SECONDS)]),
  ...(state.key === '' ? [] : ['-key', state.key]),
]

const start = (flags: readonly string[]): Promise<RunningService> =>
  new Promise((resolve, reject) => {
    const child = spawn(requireBinary(), ['-addr', '127.0.0.1:0', ...flags], { stdio: ['ignore', 'pipe', 'inherit'] })
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

  /** Makes the service answer with the status; a status other than 200 refuses every request. */
  async setStatus(name: string, status: number): Promise<void> {
    await this.restart(name, { ...this.getState(name), status })
  }

  /** Makes the service refuse a request that does not carry the key. */
  async setKey(name: string, key: string): Promise<void> {
    await this.restart(name, { ...this.getState(name), key })
  }

  getAddress(name: string): string {
    const service = this.running.get(name)
    if (service === undefined) throw new Error(`no service is named "${name}": begin with the service "${name}" answers with the status "200"`)
    return service.address
  }

  stopAll(): void {
    for (const name of [...this.running.keys()]) this.stop(name)
  }

  private getState(name: string): ServiceState {
    return this.states.get(name) ?? { status: WORKING, key: '' }
  }

  private async restart(name: string, state: ServiceState): Promise<void> {
    this.stop(name)
    this.states.set(name, state)
    this.running.set(name, await start(createServiceFlags(state)))
  }

  private stop(name: string): void {
    const service = this.running.get(name)
    this.running.delete(name)
    if (service === undefined) return
    service.child.kill()
    if (service.child.pid !== undefined) freeFromRemoval('pid', service.child.pid)
  }
}
