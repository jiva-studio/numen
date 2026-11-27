/** Controls the containers, administrative commands and database of the stand named by the test run. */
import { execFile } from 'node:child_process'
import { setTimeout as wait } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { BucketTransfer, type ComposeCommand, type ComposeOptions } from './bucketTransfer.js'

const run = promisify(execFile)

export const ENV_OWN_STACK = 'NUMEN_SYNC_OWN_STACK'
const COMPOSE_FILE = fileURLToPath(new URL('../../../server/compose.yaml', import.meta.url))
const DATABASE = 'numen_sync'
const OWNER = 'numen_sync_owner'
const DUMP = '/tmp/numen-acceptance.dump'
const GIB = 1024 ** 3
const START_LIMIT_MS = 90_000

const createCompose = (project: string): ComposeCommand =>
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
  readonly bucketTransfer: BucketTransfer

  constructor(project: string, private readonly compose = createCompose(project)) {
    this.bucketTransfer = new BucketTransfer(compose)
  }

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
    await this.bucketTransfer.finish()
    await this.startServices()
  }

  /** Returns the stand to a running state after a scenario. */
  async resetServer(): Promise<void> {
    await this.bucketTransfer.reset()
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
    await this.bucketTransfer.finish()
    await this.compose(['stop', '--timeout', '2', 'sync'])
    await this.runSQL(`DROP DATABASE IF EXISTS ${DATABASE} WITH (FORCE)`, 'postgres')
    await this.runSQL(`CREATE DATABASE ${DATABASE} OWNER ${OWNER}`, 'postgres')
    await this.compose(['exec', '-T', 'postgres', 'pg_restore', '-U', 'postgres', '--exit-on-error', '--dbname', DATABASE, DUMP])
    await this.compose(['start', 'sync'])
    await this.waitHealthy()
    if (isTold) await this.compose(['exec', '-T', 'sync', '/numen-sync', 'admin', 'issue-restore-epoch'])
  }
}
