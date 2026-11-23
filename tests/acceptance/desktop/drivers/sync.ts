/**
 * The sync server a scenario names and what the application does about it: a program of the test that replaces
 * the server on this computer, the plugin "Numen Sync" it signs in to, and the processes and warnings of
 * plugins that the application lists. The program is the fake sync server of the plugin "Numen Sync": the CI jobs
 * unpack it, and `make app_desktop_acceptance` builds it.
 */
import { execFile, spawn, type ChildProcess } from 'node:child_process'
import { once } from 'node:events'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'
import { expect } from 'playwright/test'
import { Gate } from './gate.js'
import type { IdentityDriver } from './identity.js'
import { getDeviceFingerprint } from './link.js'
import { freeFromRemoval, holdForRemoval } from './reaper.js'
import type { Scenario } from './scenario.js'

const PLUGIN = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../plugins/tools-sync')
const BINARY = path.join(PLUGIN, 'bin', 'fakesync')
const SYNC_PLUGIN = 'md.numen.tools.sync'
/** The server lets a session live this long, so the plugin signs in again 2 seconds after it signed in. */
const SESSION_SECONDS = 62
/** How often the plugin asks the server for changes, so a scenario waits seconds and not half a minute. */
const SYNC_INTERVAL_SECONDS = '1'
/** How long the plugin watches a removed file before it sends the removal, so a scenario waits a second and not a minute. */
const REMOVAL_DELAY_SECONDS = '1'
/** A stream of change hints is silent for at most this long, so a connection that half-dropped shows within a scenario. */
const HEARTBEAT = '1s'
/** A subscription that ended fewer days ago than this is in the grace period; the server allows writes then. */
const GRACE_DAYS = 7
/** A subscription that ended this many days ago or more is deleted. */
const RETENTION_DAYS = 60
/** A protocol version no build of the application implements. */
const NEWER_PROTOCOL = 1000
/** The plugin tries to sign in within this time of its start; a server that has no session after it has none to give. */
const SETTLE_MS = 3_000
/** The computer that reaches the server first commits its changes within a few passes, which come every second. */
const LEAD_MS = 5_000

/** A number and a unit, as a scenario writes an interval. */
const INTERVAL_PATTERN = /^(\d+) (second|minute|hour)s?$/
const INTERVAL_UNITS = { second: 1, minute: 60, hour: 3600 } as const
/** The server counts the streams of the plugin on both computers; a change arrives within a few seconds of retrying. */
const STREAM_TIMEOUT_MS = 15_000

/** The plugins a scenario names, by the name a person reads: the id of the manifest and the file of the backend. */
const PLUGINS: Readonly<Partial<Record<string, { readonly id: string; readonly backend: string }>>> = {
  'Numen Sync': { id: SYNC_PLUGIN, backend: 'tools-sync' },
}

interface SessionRow {
  readonly deviceSignKey: string
  readonly accountFingerprint: string
  readonly token: string
}

interface ProcessRow {
  readonly pluginId?: string
  readonly startTimeUnixMs?: string
}

interface RunningSync {
  readonly child: ChildProcess
  readonly address: string
}

const run = promisify(execFile)

const requireBinary = (): string => {
  if (!fs.existsSync(BINARY)) {
    throw new Error(`${BINARY} is missing; build it with make app_desktop_acceptance`)
  }
  return BINARY
}

const findPlugin = (name: string): { readonly id: string; readonly backend: string } => {
  const found = PLUGINS[name]
  if (found === undefined) throw new Error(`"${name}" is not a plugin the scenarios know: use ${Object.keys(PLUGINS).join(', ')}`)
  return found
}

const startProgram = (listen: string, flags: readonly string[]): Promise<RunningSync> =>
  new Promise((resolve, reject) => {
    const child = spawn(requireBinary(), ['-addr', listen, '-session-ttl', `${SESSION_SECONDS}s`, ...flags], {
      stdio: ['ignore', 'pipe', 'inherit'],
    })
    if (child.pid !== undefined) holdForRemoval('pid', child.pid)
    child.once('error', reject)
    child.once('exit', () => reject(new Error('the sync server ended before it listened')))
    child.stdout.once('data', (chunk: Buffer) => {
      child.removeAllListeners('exit')
      resolve({ child, address: chunk.toString().trim() })
    })
  })

/** The seconds a scenario names as an interval: "1 hour" is 3600 and "1 second" is 1. */
export const getIntervalSeconds = (text: string): number => {
  const found = INTERVAL_PATTERN.exec(text)
  const unit = found?.[2]
  if (found?.[1] === undefined || unit === undefined) {
    throw new Error(`"${text}" does not name an interval: write a number and a unit, such as "1 hour"`)
  }
  return Number(found[1]) * INTERVAL_UNITS[unit as keyof typeof INTERVAL_UNITS]
}

/** The phase the stand-in of the payment sets for a subscription that ended some days ago. */
export const getSubscriptionPhase = (daysAgo: number): string => {
  if (daysAgo < GRACE_DAYS) return 'grace'
  return daysAgo < RETENTION_DAYS ? 'read_only' : 'deleted'
}

/** A plan a scenario names: the name a person reads and its size in GB. */
export interface PlanSpec {
  readonly name: string
  readonly gigabytes: number
}

const BYTES_IN_GB = 2 ** 30

const PLAN_PATTERN = /"([^"]+)" with (\d+) GB/g

/** The plans a step names, such as `"Basic" with 10 GB, "Plus" with 100 GB and "Pro" with 250 GB`, in the order of the step. */
export const parsePlans = (text: string): PlanSpec[] =>
  [...text.matchAll(PLAN_PATTERN)].map((one) => ({ name: one[1], gigabytes: Number(one[2]) }))

/** The value of the `-plans` flag of the fake sync server: the id of each plan (its name in lower case) and its size in GB. */
export const getPlansFlag = (plans: readonly PlanSpec[]): string =>
  plans.map((one) => `${one.name.toLowerCase()}:${one.gigabytes}`).join(',')

/** The bytes the cloud holds when it holds some GB of the notes. */
export const getCloudBytes = (gigabytes: number): number => gigabytes * BYTES_IN_GB

/** The days a trial of a plan runs on the stand-in of the payment. */
export const TRIAL_DAYS = 14

/** The ids of the processes whose program is the file of the backend of a plugin, as the system lists them. */
export const findBackendPids = (processes: string, backend: string): number[] =>
  processes
    .split('\n')
    .map((line) => /^\s*(\d+)\s+(\S+)/.exec(line))
    .flatMap((found) => (found?.[1] !== undefined && found[2].endsWith(`/bin/${backend}`) ? [Number(found[1])] : []))

export class SyncDriver {
  private running: RunningSync | undefined
  private extraFlags: readonly string[] = []
  private stateDir: string | undefined
  private listen: string | undefined
  private heldGigabytes = 0
  private trial: { readonly plan: string; readonly endsAt: Date } | undefined
  private readonly gates = new Map<Scenario, Gate>()
  private held: readonly Gate[] = []

  constructor(
    private readonly scenario: Scenario,
    private readonly identity: IdentityDriver,
  ) {}

  /** Starts the sync server that sells subscriptions: an account has none until it pays on the page the server serves. */
  async startSellingSubscriptions(): Promise<void> {
    this.extraFlags = ['-subscriptions']
    await this.start()
  }

  /** Starts the sync server that sells the plans: an account has none until it pays for one on the page the server serves. */
  async startSellingPlans(plans: readonly PlanSpec[]): Promise<void> {
    this.extraFlags = ['-subscriptions', '-plans', getPlansFlag(plans)]
    await this.start()
  }

  /** Starts a trial of the plan on the account, as the stand-in of the payment does for an account that asked for one. */
  async startTrial(plan: string): Promise<void> {
    await this.expectSessionForAccount()
    await this.sendAdminRequest(`trial?plan=${plan.toLowerCase()}`, 'the start of the trial')
    this.trial = { plan, endsAt: new Date(Date.now() + TRIAL_DAYS * 24 * 60 * 60 * 1000) }
  }

  /** Cancels the trial the account runs: the subscription stays until the trial ends. */
  async cancelTrial(): Promise<void> {
    if (this.trial === undefined) throw new Error('no trial started: begin with the person starts a trial of a plan')
    await this.sendAdminRequest(`cancel-trial?plan=${this.trial.plan.toLowerCase()}`, 'the cancel of the trial')
  }

  /** The day the trial that started last ends. */
  getTrialEnd(): Date {
    if (this.trial === undefined) throw new Error('no trial started: begin with the person starts a trial of a plan')
    return this.trial.endsAt
  }

  private async sendAdminRequest(request: string, what: string): Promise<void> {
    const answer = await fetch(`${this.requireRunning().address}/admin/${request}`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to ${what}`)
  }

  /** Makes the cloud hold some GB of the notes of the person. */
  async holdInCloud(gigabytes: number): Promise<void> {
    this.heldGigabytes = gigabytes
    await this.setCloudBytes()
  }

  /** Deletes some GB of the notes of the person from the cloud. */
  async deleteFromCloud(gigabytes: number): Promise<void> {
    if (gigabytes > this.heldGigabytes) throw new Error(`the cloud holds ${this.heldGigabytes} GB: ${gigabytes} GB cannot be deleted`)
    this.heldGigabytes -= gigabytes
    await this.setCloudBytes()
  }

  private async setCloudBytes(): Promise<void> {
    const answer = await fetch(`${this.requireRunning().address}/admin/usage?bytes=${getCloudBytes(this.heldGigabytes)}`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the bytes the cloud holds`)
  }


  /** Sets the subscription of every account to one that ended some days ago. */
  async endSubscription(daysAgo: number): Promise<void> {
    const phase = getSubscriptionPhase(daysAgo)
    const answer = await fetch(`${this.requireRunning().address}/admin/subscription?phase=${phase}&days=${daysAgo}`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the end of the subscription`)
    if (phase === 'deleted') await this.sendAdminRequest('/admin/retention', 'the deletion of the vaults')
  }

  /** Starts the sync server and tells the plugin its address, before the application starts. The server keeps its state in a folder, so it can stop and run again. */
  async start(): Promise<void> {
    this.stateDir = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-sync-'))
    holdForRemoval('dir', this.stateDir)
    this.running = await startProgram('127.0.0.1:0', this.getStateFlags())
    this.listen = new URL(this.running.address).host
    await this.setPlugin(this.scenario)
  }

  /** Tells the plugin of a scenario where the sync server is and how often to ask it. The plugin reaches the server through a gate of its computer. */
  async setPlugin(scenario: Scenario): Promise<void> {
    const gate = await Gate.listen(() => this.getTarget())
    this.gates.set(scenario, gate)
    scenario.setPluginSetting(SYNC_PLUGIN, 'server', gate.getAddress())
    scenario.setPluginSetting(SYNC_PLUGIN, 'syncIntervalSeconds', SYNC_INTERVAL_SECONDS)
    scenario.setPluginSetting(SYNC_PLUGIN, 'removalDelaySeconds', REMOVAL_DELAY_SECONDS)
  }

  /** Makes the other computers unable to reach the sync server until it runs again and this computer had time to send its changes. */
  holdOthers(lead: Scenario): void {
    this.held = [...this.gates].filter(([scenario]) => scenario !== lead).map(([, gate]) => gate)
    for (const gate of this.held) gate.close()
  }

  /** Makes the sync server unreachable from the computer of the scenario, or reachable again. */
  setReachable(scenario: Scenario, isReachable: boolean): void {
    const gate = this.gates.get(scenario)
    if (gate === undefined) throw new Error('the computer has no way to the sync server')
    if (isReachable) gate.reopen()
    else gate.close()
  }

  private getTarget(): { readonly host: string; readonly port: number } {
    if (this.listen === undefined) throw new Error('no sync server ran: begin with a sync server runs')
    const url = new URL(`http://${this.listen}`)
    return { host: url.hostname, port: Number(url.port) }
  }

  /** Keeps a copy of everything the server holds now, as the operator's backup does. */
  async makeBackup(): Promise<void> {
    await this.sendAdminRequest('/admin/backup', 'the backup')
  }

  /** Puts the server back to the copy of the backup: what it received after is gone, and the computers are not told. */
  async restoreFromBackup(): Promise<void> {
    await this.sendAdminRequest('/admin/restore', 'the restore')
  }

  /** Tells the computers that the server was restored. */
  async announceRestore(): Promise<void> {
    await this.sendAdminRequest('/admin/announce', 'the announcement of the restore')
  }

  private async sendAdminRequest(path: string, what: string): Promise<void> {
    const answer = await fetch(`${this.requireRunning().address}${path}`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to ${what}`)
  }

  /** Ends the sync server and keeps its state and its address, as a server that goes away does. */
  async pause(): Promise<void> {
    const running = this.requireRunning()
    this.stop()
    await once(running.child, 'exit')
  }

  /** Makes the sync server end itself once it received (the parts of a file going up) or sent (the body of a blob going down) this many bytes after this call. */
  async breakAfter(direction: 'received' | 'sent', bytes: number): Promise<void> {
    const call = direction === 'received' ? 'stop-after-bytes' : 'stop-after-download-bytes'
    const answer = await fetch(`${this.requireRunning().address}/admin/${call}?n=${bytes}`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the break of the connection`)
  }

  /** Runs the sync server again at the address it had, over the state it kept; a server that is still to break is waited for first. */
  async resume(): Promise<void> {
    if (this.listen === undefined) throw new Error('no sync server ran: begin with a sync server runs')
    const previous = this.running
    if (previous !== undefined && previous.child.exitCode === null && previous.child.signalCode === null) {
      await once(previous.child, 'exit')
    }
    this.running = await startProgram(this.listen, this.getStateFlags())
    if (this.held.length > 0) {
      await new Promise((resolve) => setTimeout(resolve, LEAD_MS))
      for (const gate of this.held) gate.reopen()
      this.held = []
    }
  }

  /** Makes the server drop the removals it keeps, as it does for removals older than its retention. */
  async forgetRemovals(): Promise<void> {
    const answer = await fetch(`${this.requireRunning().address}/admin/prune-tombstones`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the forgetting of removals`)
  }

  /** Reads what the server stores of every blob and expects the text to be in none of it. */
  async expectNoStoredText(text: string): Promise<void> {
    const answer = await fetch(`${this.requireRunning().address}/admin/blobs`)
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the list of its blobs`)
    const stored = Buffer.from(await answer.arrayBuffer())
    expect(stored.length, 'the server stores no blob yet').toBeGreaterThan(0)
    expect(stored.includes(Buffer.from(text, 'utf8'))).toBe(false)
  }

  private getStateFlags(): string[] {
    const state = this.stateDir === undefined ? [] : ['-state-dir', this.stateDir]
    return [...state, '-heartbeat', HEARTBEAT, ...this.extraFlags]
  }

  /** Makes the sync server need a protocol no application implements, where it listened before. */
  async needNewerProtocol(): Promise<void> {
    const previous = this.requireRunning()
    this.stop()
    await once(previous.child, 'exit')
    const listen = new URL(previous.address).host
    this.running = await startProgram(listen, [...this.getStateFlags(), '-protocol', String(NEWER_PROTOCOL)])
  }

  stop(): void {
    const running = this.running
    this.running = undefined
    if (running === undefined) return
    running.child.kill()
    if (running.child.pid !== undefined) freeFromRemoval('pid', running.child.pid)
  }

  /** Ends the server for good and removes the folder of its state. */
  remove(): void {
    for (const gate of this.gates.values()) gate.stop()
    this.gates.clear()
    this.stop()
    if (this.stateDir === undefined) return
    fs.rmSync(this.stateDir, { recursive: true, force: true })
    freeFromRemoval('dir', this.stateDir)
    this.stateDir = undefined
  }

  /** Ends every open stream of change hints at the server, as a connection that drops does, and waits until none is open. */
  async dropStreams(): Promise<void> {
    const answer = await fetch(`${this.requireRunning().address}/admin/streams/end`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the end of the streams`)
    await this.expectStreams(0)
  }

  /** Waits until the server holds exactly this many open streams of change hints. */
  async expectStreams(count: number): Promise<void> {
    await expect.poll(() => this.readStreamCount(), { timeout: STREAM_TIMEOUT_MS }).toBe(count)
  }

  async expectSession(): Promise<void> {
    await expect.poll(async () => (await this.readSessions()).length, { timeout: 15_000 }).toBeGreaterThan(0)
  }

  private async readSessionDevices(): Promise<string[]> {
    return (await this.readSessions()).map((one) => getDeviceFingerprint(one.deviceSignKey))
  }

  async expectSessionOfDevice(fingerprint: string): Promise<void> {
    await expect.poll(() => this.readSessionDevices(), { timeout: 15_000 }).toContain(fingerprint)
  }

  async expectNoSessionOfDevice(fingerprint: string): Promise<void> {
    await expect.poll(() => this.readSessionDevices(), { timeout: 15_000 }).not.toContain(fingerprint)
  }

  async expectSessionForAccount(): Promise<void> {
    const account = this.identity.getAccountFingerprint()
    await expect
      .poll(async () => (await this.readSessions()).map((one) => one.accountFingerprint), { timeout: 15_000 })
      .toContain(account)
  }

  async expectNoSession(): Promise<void> {
    await new Promise((resolve) => setTimeout(resolve, SETTLE_MS))
    expect(await this.readSessions()).toEqual([])
  }

  /** Ends every session of the server, so a session it lists afterwards is one the plugin opened again. */
  async endSession(): Promise<void> {
    const answer = await fetch(`${this.requireRunning().address}/admin/sessions/end`, { method: 'POST' })
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the end of the sessions`)
  }

  async expectRunning(plugin: string, isRunning: boolean): Promise<void> {
    const { id } = findPlugin(plugin)
    await expect
      .poll(async () => (await this.readProcesses()).some((one) => one.pluginId === id), { message: `the plugin "${plugin}" runs`, timeout: 40_000 })
      .toBe(isRunning)
  }

  /** Ends the process of the backend of the plugin the way the system would, and waits until the list of processes no longer shows that process. */
  async endProcess(plugin: string): Promise<void> {
    const { id, backend } = findPlugin(plugin)
    const before = (await this.readProcesses()).find((one) => one.pluginId === id)
    const listed = await run('ps', ['-A', '-o', 'pid=,args='])
    const pids = findBackendPids(listed.stdout, backend)
    if (pids.length === 0) throw new Error(`no process of the backend of the plugin "${plugin}" runs`)
    for (const pid of pids) process.kill(pid, 'SIGKILL')
    await expect
      .poll(async () => {
        const now = (await this.readProcesses()).find((one) => one.pluginId === id)
        return now === undefined || now.startTimeUnixMs !== before?.startTimeUnixMs
      })
      .toBe(true)
  }

  async expectWarning(plugin: string, text: string): Promise<void> {
    const { id } = findPlugin(plugin)
    await expect
      .poll(
        async () => {
          const head = await this.scenario.getServer().readStreamHead('PluginHealthService/WatchHealth', {})
          const items = (head['items'] ?? []) as { pluginId?: string; text?: string }[]
          return items.filter((one) => one.pluginId === id).map((one) => one.text)
        },
        { timeout: 15_000 },
      )
      .toContain(text)
  }

  private requireRunning(): RunningSync {
    if (this.running === undefined) throw new Error('no sync server runs: begin with a sync server runs')
    return this.running
  }

  private async readSessions(): Promise<SessionRow[]> {
    const answer = await fetch(`${this.requireRunning().address}/admin/sessions`)
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the list of its sessions`)
    return (await answer.json()) as SessionRow[]
  }

  private async readStreamCount(): Promise<number> {
    const answer = await fetch(`${this.requireRunning().address}/admin/streams`)
    if (!answer.ok) throw new Error(`the sync server answered ${answer.status} to the count of its streams`)
    return Number((await answer.text()).trim())
  }

  private async readProcesses(): Promise<ProcessRow[]> {
    const head = await this.scenario.getServer().readStreamHead('PluginProcessService/WatchPluginProcesses', {})
    return (head['processes'] ?? []) as ProcessRow[]
  }
}
