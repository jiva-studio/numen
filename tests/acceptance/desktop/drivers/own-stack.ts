/** Controls the containers, administrative commands and database of the stand named by the test run. */
import { execFile } from 'node:child_process'
import { setTimeout as wait } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'

const run = promisify(execFile)

export const ENV_OWN_STACK = 'NUMEN_SYNC_OWN_STACK'
const COMPOSE_FILE = fileURLToPath(new URL('../../../server/compose.yaml', import.meta.url))
const DATABASE = 'numen_sync'
const OWNER = 'numen_sync_owner'
const DUMP = '/tmp/numen-acceptance.dump'
const GIB = 1024 ** 3
const START_LIMIT_MS = 90_000
const BREAK_LIMIT_MS = 30_000

interface TrafficWatch {
  readonly controller: AbortController
  readonly result: Promise<Error | undefined>
}

interface Traffic {
  readonly received: number
  readonly sent: number
}

interface ComposeOptions {
  readonly timeout?: number
  readonly signal?: AbortSignal
}

const createCompose = (project: string) =>
  async (args: readonly string[], stdin?: string, options: ComposeOptions = {}): Promise<string> => {
    const child = run('docker', ['compose', '-f', COMPOSE_FILE, '-p', project, ...args], {
      timeout: options.timeout ?? 120_000,
      signal: options.signal,
      maxBuffer: 16 * 1024 * 1024,
    })
    child.child.stdin?.end(stdin)
    return (await child).stdout
  }

export class OwnStack {
  private watcher: TrafficWatch | undefined

  constructor(project: string, private readonly compose = createCompose(project)) {}

  private async runSQL(statements: string, database = DATABASE): Promise<string> {
    return this.compose(['exec', '-T', 'postgres', 'psql', '-U', 'postgres', '--set', 'ON_ERROR_STOP=1', '--quiet', '-At', '--dbname', database], statements)
  }

  private async waitHealthy(): Promise<void> {
    const deadline = Date.now() + START_LIMIT_MS
    for (;;) {
      try {
        await this.compose(['exec', '-T', 'sync', '/numen-sync', 'healthcheck'], undefined, { timeout: 10_000 })
        return
      } catch (error) {
        if (Date.now() > deadline) throw new Error('the sync server is not healthy after the start', { cause: error })
        await wait(250)
      }
    }
  }

  /** Stops the server: every open stream ends with it. */
  async stopServer(): Promise<void> {
    await this.compose(['stop', '--timeout', '2', 'sync'])
  }

  /** Starts the server, and the bucket when a break stopped it, and waits until the server is healthy. */
  async startServer(): Promise<void> {
    await this.waitForInterruption()
    this.watcher = undefined
    await this.startServices()
  }

  /** Returns the stand to a running state after a scenario. */
  async resetServer(): Promise<void> {
    await this.stopTrafficWatch()
    await this.startServices()
  }

  private async startServices(): Promise<void> {
    await this.compose(['start', 'bucket', 'sync'])
    await this.waitHealthy()
  }

  async restartServer(): Promise<void> {
    await this.stopServer()
    await this.startServer()
  }

  /** Ends the sessions of every account of the stack. */
  async endSessions(): Promise<void> {
    const accounts = (await this.runSQL('SELECT encode(id, \'hex\') FROM accounts')).split('\n').filter((line) => line !== '')
    for (const account of accounts) await this.compose(['exec', '-T', 'sync', '/numen-sync', 'admin', 'revoke-sessions', '--account', account])
  }

  /** Moves the end of the paid period of every account to the days ago; the grace the plan has stays after it. */
  async endSubscriptions(daysAgo: number): Promise<void> {
    await this.runSQL(
      `UPDATE entitlements SET period_end = now() - interval '${daysAgo} days', ends_at = now() - interval '${daysAgo} days' + grace_days * interval '1 day'`,
    )
  }

  /** Sets the bytes the cloud holds for every account. */
  async setUsage(gigabytes: number): Promise<void> {
    await this.runSQL(
      `INSERT INTO account_usage (account_id, used_bytes) SELECT id, ${Math.round(gigabytes * GIB)} FROM accounts ON CONFLICT (account_id) DO UPDATE SET used_bytes = EXCLUDED.used_bytes`,
    )
  }

  async addUsage(gigabytes: number): Promise<void> {
    await this.runSQL(`UPDATE account_usage SET used_bytes = greatest(used_bytes + ${Math.round(gigabytes * GIB)}, 0)`)
  }

  /** Ages every deletion record past its life and has the server prune them. */
  async forgetRemovals(): Promise<void> {
    await this.runSQL("UPDATE files SET committed_at = now() - interval '91 days' WHERE deleted")
    await this.compose(['exec', '-T', 'sync', '/numen-sync', 'admin', 'prune-tombstones'])
  }

  async makeBackup(): Promise<void> {
    await this.compose(['exec', '-T', 'postgres', 'pg_dump', '-U', 'postgres', '--format=custom', '--file', DUMP, DATABASE])
  }

  /** Replaces the database with the backup, as an operator does, and tells the computers by a restore epoch when the step says so. */
  async restoreBackup(isTold: boolean): Promise<void> {
    await this.waitForInterruption()
    this.watcher = undefined
    await this.compose(['stop', '--timeout', '2', 'sync'])
    await this.runSQL(`DROP DATABASE IF EXISTS ${DATABASE} WITH (FORCE)`, 'postgres')
    await this.runSQL(`CREATE DATABASE ${DATABASE} OWNER ${OWNER}`, 'postgres')
    await this.compose(['exec', '-T', 'postgres', 'pg_restore', '-U', 'postgres', '--exit-on-error', '--dbname', DATABASE, DUMP])
    await this.compose(['start', 'sync'])
    await this.waitHealthy()
    if (isTold) await this.compose(['exec', '-T', 'sync', '/numen-sync', 'admin', 'issue-restore-epoch'])
  }

  private async readTraffic(signal?: AbortSignal): Promise<Traffic> {
    const text = await this.compose(['exec', '-T', 'bucket', 'cat', '/proc/net/dev'], undefined, { signal, timeout: 10_000 })
    const line = text.split('\n').find((one) => one.trim().startsWith('eth0:'))
    const fields = line?.split(':')[1]?.trim().split(/\s+/).map(Number)
    const received = fields?.[0]
    const sent = fields?.[8]
    if (received === undefined || sent === undefined || !Number.isFinite(received) || !Number.isFinite(sent)) {
      throw new Error('the bucket container shows no eth0 traffic')
    }
    return { received, sent }
  }

  /** Stops the bucket once the bytes of a direction passed the limit, as seen from the bucket: received is what went up, sent what went down. */
  async breakAfter(direction: 'received' | 'sent', bytes: number): Promise<void> {
    if (this.watcher !== undefined) throw new Error('the stand already waits for a transfer interruption')
    const before = await this.readTraffic()
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(new Error('the transfer did not break within 30 seconds')), BREAK_LIMIT_MS)
    const result = this.watchTraffic(direction, before[direction] + bytes, controller.signal)
      .catch((cause: unknown) => {
        const reason: unknown = controller.signal.reason
        return reason instanceof Error ? reason : new Error('the bucket traffic could not be interrupted', { cause })
      })
      .finally(() => clearTimeout(timeout))
    this.watcher = { controller, result }
  }

  async waitForInterruption(): Promise<void> {
    const error = await this.watcher?.result
    if (error !== undefined) throw error
  }

  private async watchTraffic(direction: keyof Traffic, limit: number, signal: AbortSignal): Promise<undefined> {
    for (;;) {
      const traffic = await this.readTraffic(signal)
      signal.throwIfAborted()
      if (traffic[direction] >= limit) {
        await this.compose(['kill', 'bucket'], undefined, { signal })
        return
      }
    }
  }

  private async stopTrafficWatch(): Promise<void> {
    const watcher = this.watcher
    if (watcher === undefined) return
    this.watcher = undefined
    watcher.controller.abort(new Error('the scenario ended before the transfer broke'))
    await watcher.result
  }
}
