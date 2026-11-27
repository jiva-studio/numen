/**
 * The real sync server a scenario names and what the application does about it: the address and the root key the plugin "Numen Sync" gets,
 * the account of the scenario on that server, and the processes and warnings of plugins that the application lists. The server is the
 * stack of tests/server or the stand of the owner, shared by every scenario; a power over it that a shared server cannot give is
 * the mark of a scenario tagged @own-server.
 */
import { execFile, execFileSync } from 'node:child_process'
import { promisify } from 'node:util'
import { expect } from 'playwright/test'
import { removeSeed, makeSeed, type IdentitySeed } from './identity-seed.js'
import type { IdentityDriver } from './identity.js'
import { expectTextNotStored, readDevices, readPlans, requireStack, type SyncStack } from './sync-stack.js'
import type { Scenario } from './scenario.js'
import { ENV_OWN_STACK, OwnStack } from './own-stack.js'

export const SYNC_PLUGIN = 'md.numen.tools.sync'
/** The address the plugin signs in to is the one the server names as its own, and the key it trusts for the epochs of that server. */
const SETTING_SERVER = 'server'
const SETTING_ROOT_KEY = 'epochRootKey'
/** How often the plugin asks the server for changes, so a scenario waits seconds and not half a minute. */
const SYNC_INTERVAL_SECONDS = '1'
/** How long the plugin watches a removed file before it sends the removal, so a scenario waits a second and not a minute. */
const REMOVAL_DELAY_SECONDS = '1'
/** The plugin asks the server every second, so every computer ran several passes in this time; a note that has not arrived by then does not arrive. */
const SETTLE_MS = 6_000
/** A device the plugin added to the chain, or a revoked one, shows on the server within a few passes. */
const DEVICE_LIST_TIMEOUT_MS = 15_000
/** How long the lead computer has the server to itself before the held computers go on. */
const HOLD_MS = 6_000

/** A number and a unit, as a scenario writes an interval. */
const INTERVAL_PATTERN = /^(\d+) (second|minute|hour)s?$/
const INTERVAL_UNITS = { second: 1, minute: 60, hour: 3600 } as const

/** The plugins a scenario names, by the name a person reads: the id of the manifest and the file of the backend. */
const PLUGINS: Readonly<Partial<Record<string, { readonly id: string; readonly backend: string }>>> = {
  'Numen Sync': { id: SYNC_PLUGIN, backend: 'tools-sync' },
}

interface ProcessRow {
  readonly pluginId?: string
  readonly startTimeUnixMs?: string
}

const run = promisify(execFile)

const findPlugin = (name: string): { readonly id: string; readonly backend: string } => {
  const found = PLUGINS[name]
  if (found === undefined) throw new Error(`"${name}" is not a plugin the scenarios know: use ${Object.keys(PLUGINS).join(', ')}`)
  return found
}

/** The seconds a scenario names as an interval: "1 hour" is 3600 and "1 second" is 1. */
export const getIntervalSeconds = (text: string): number => {
  const found = INTERVAL_PATTERN.exec(text)
  const unit = found?.[2]
  if (found?.[1] === undefined || unit === undefined) {
    throw new Error(`"${text}" does not name an interval: write a number and a unit, such as "1 hour"`)
  }
  return Number(found[1]) * INTERVAL_UNITS[unit as keyof typeof INTERVAL_UNITS]
}

/** A plan a scenario names: the name a person reads and its size in GB. */
export interface PlanSpec {
  readonly name: string
  readonly gigabytes: number
}

const PLAN_PATTERN = /"([^"]+)" with (\d+) GB/g

/** The plans a step names, such as `"Basic" with 10 GB, "Plus" with 100 GB and "Pro" with 250 GB`, in the order of the step. */
export const parsePlans = (text: string): PlanSpec[] =>
  [...text.matchAll(PLAN_PATTERN)].map((one) => ({ name: one[1], gigabytes: Number(one[2]) }))

/** The ids of the processes whose program is the file of the backend of a plugin, as the system lists them. */
export const findBackendPids = (processes: string, backend: string): number[] =>
  processes
    .split('\n')
    .map((line) => /^\s*(\d+)\s+(\S+)/.exec(line))
    .flatMap((found) => (found?.[1] !== undefined && found[2].endsWith(`/bin/${backend}`) ? [Number(found[1])] : []))

export class SyncDriver {
  private stack: SyncStack | undefined
  private startedAt = new Date()
  private own: OwnStack | undefined
  private held: Scenario[] = []
  private seeded: { readonly seed: IdentitySeed; readonly folder: string } | undefined

  constructor(
    private readonly scenario: Scenario,
    private readonly identity: IdentityDriver,
  ) {}

  /** Names the real sync server to the plugin, before the application starts. */
  start(): void {
    this.stack = requireStack()
    this.startedAt = new Date()
    this.setPlugin(this.scenario)
  }

  isStarted(): boolean {
    return this.stack !== undefined
  }

  /** Names the server that sells subscriptions: the stand-in of the payment of the stack. */
  startSellingSubscriptions(): void {
    this.start()
  }

  /** Names the server that sells the plans, and expects the stand-in of the payment to sell those, by name and size. */
  async startSellingPlans(plans: readonly PlanSpec[]): Promise<void> {
    this.start()
    const sold = await readPlans(this.requireStack())
    for (const plan of plans) {
      expect(sold, `the stand-in sells the plan "${plan.name}" with ${plan.gigabytes} GB`).toContainEqual(plan)
    }
  }

  /** Makes the account of the scenario on the server, with the keys of three computers, and buys a plan at the stand-in when one is named. */
  async makeAccount(plan: string | undefined): Promise<IdentitySeed> {
    this.seeded = await makeSeed(this.requireStack(), plan)
    return this.seeded.seed
  }

  /** The account made for the scenario. */
  getSeed(): IdentitySeed {
    if (this.seeded === undefined) throw new Error('no account was made: begin with the person has an identity')
    return this.seeded.seed
  }

  /** Tells the plugin of a scenario where the sync server is, which key vouches for its epochs and how often to ask it. */
  setPlugin(scenario: Scenario): void {
    const stack = this.requireStack()
    scenario.setPluginSetting(SYNC_PLUGIN, SETTING_SERVER, stack.url)
    scenario.setPluginSetting(SYNC_PLUGIN, SETTING_ROOT_KEY, stack.rootKey)
    scenario.setPluginSetting(SYNC_PLUGIN, 'syncIntervalSeconds', SYNC_INTERVAL_SECONDS)
    scenario.setPluginSetting(SYNC_PLUGIN, 'removalDelaySeconds', REMOVAL_DELAY_SECONDS)
  }

  /** Reads what the server stores of every blob that came after the scenario began and expects the text to be in none of it. */
  async expectNoStoredText(text: string): Promise<void> {
    await expectTextNotStored(this.requireStack(), text, this.startedAt)
  }

  private async readDevices(): Promise<string[]> {
    return readDevices(this.requireStack(), this.identity.getWords())
  }

  async expectSessionOfDevice(fingerprint: string): Promise<void> {
    await expect.poll(() => this.readDevices(), { timeout: DEVICE_LIST_TIMEOUT_MS }).toContain(fingerprint)
  }

  async expectNoSessionOfDevice(fingerprint: string): Promise<void> {
    await expect.poll(() => this.readDevices(), { timeout: DEVICE_LIST_TIMEOUT_MS }).not.toContain(fingerprint)
  }

  /** Expects the account of the identity on the server: the first sign-in of a device makes it. */
  async expectSignedIn(): Promise<void> {
    await expect.poll(async () => (await this.readDevices()).length, { timeout: DEVICE_LIST_TIMEOUT_MS }).toBeGreaterThan(0)
  }

  /** Expects the server to know no account of the identity. The step before it names the warning the plugin shows when it does not sign in, so the plugin has tried by now. */
  async expectNotSignedIn(): Promise<void> {
    await this.expectSettled()
    if (this.identity.hasWords()) expect(await this.readDevices()).toEqual([])
  }

  /** Waits so long that every computer ran passes of its own, twice over: the first round sends what the computers hold, the second brings it to the others. */
  async expectSettled(): Promise<void> {
    await new Promise((resolve) => setTimeout(resolve, SETTLE_MS))
  }

  /** Waits so long that the stream of change hints of every computer is open again. */
  async expectStreams(_count: number): Promise<void> {
    await this.expectSettled()
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

  /** Removes the files of the account of the scenario. */
  remove(): void {
    this.own?.stopWatching()
    if (this.own !== undefined) execFileSync('docker', ['compose', '-p', this.own.project, 'start', 'bucket', 'sync'], { timeout: 120_000 })
    if (this.seeded === undefined) return
    removeSeed(this.seeded.folder)
    this.seeded = undefined
  }

  private requireStack(): SyncStack {
    if (this.stack === undefined) throw new Error('no sync server named: begin with a sync server runs')
    return this.stack
  }

  private async readProcesses(): Promise<ProcessRow[]> {
    const head = await this.scenario.getServer().readStreamHead('PluginProcessService/WatchPluginProcesses', {})
    return (head['processes'] ?? []) as ProcessRow[]
  }

  private createPowerError(power: string): Error {
    return new Error(`${power}: the sync server of this run is shared by every scenario and gives no such power; run the scenario on a stack of its own (tag @own-server)`)
  }

  private requireOwn(power: string): OwnStack {
    const project = process.env[ENV_OWN_STACK]
    if (project === undefined || project === '') throw this.createPowerError(power)
    this.own ??= new OwnStack(project)
    return this.own
  }

  /** The processes of the backend of Numen Sync that the application of a scenario started. */
  private async findPluginPids(scenario: Scenario): Promise<number[]> {
    const root = scenario.getServer().getPid()
    if (root === undefined) return []
    const listed = (await run('ps', ['-A', '-o', 'pid=,ppid=,args='])).stdout
    const rows = listed.split('\n').flatMap((line) => {
      const found = /^\s*(\d+)\s+(\d+)\s+(.*)$/.exec(line)
      return found === null ? [] : [{ pid: Number(found[1]), parent: Number(found[2]), args: found[3] }]
    })
    const family = new Set<number>([root])
    for (let grew = true; grew; ) {
      grew = false
      for (const row of rows) if (family.has(row.parent) && !family.has(row.pid)) (family.add(row.pid), (grew = true))
    }
    return rows.filter((row) => family.has(row.pid) && row.args.split(' ')[0].endsWith('/bin/tools-sync')).map((row) => row.pid)
  }

  private async signalPlugin(scenario: Scenario, signal: 'SIGSTOP' | 'SIGCONT'): Promise<void> {
    for (const pid of await this.findPluginPids(scenario)) process.kill(pid, signal)
  }

  private releaseHeld(): Promise<unknown> {
    const held = this.held.splice(0)
    return Promise.all(held.map((scenario) => this.signalPlugin(scenario, 'SIGCONT')))
  }

  startTrial(_plan: string): Promise<never> {
    return Promise.reject(this.createPowerError('start a trial of a plan for one account'))
  }

  cancelTrial(): Promise<never> {
    return Promise.reject(this.createPowerError('cancel the trial of one account'))
  }

  getTrialEnd(): never {
    throw this.createPowerError('read the end of the trial of one account')
  }

  async holdInCloud(gigabytes: number): Promise<void> {
    await this.requireOwn('make the cloud hold gigabytes of notes').setUsage(gigabytes)
  }

  async deleteFromCloud(gigabytes: number): Promise<void> {
    await this.requireOwn('delete gigabytes of notes from the cloud').addUsage(-gigabytes)
  }

  async endSubscription(daysAgo: number): Promise<void> {
    await this.requireOwn('end the subscription of one account some days ago').endSubscriptions(daysAgo)
  }

  /** Stops the plugin of every computer but the lead; the server runs again first for the lead, and the others go on after it sent. */
  async holdOthers(lead: Scenario): Promise<void> {
    this.requireOwn('cut the other computers off from the server')
    const others = [this.scenario].filter((one) => one !== lead)
    for (const one of others) await this.signalPlugin(one, 'SIGSTOP')
    this.held.push(...others)
  }

  async setReachable(scenario: Scenario, isReachable: boolean): Promise<void> {
    this.requireOwn('cut one computer off from the server')
    await this.signalPlugin(scenario, isReachable ? 'SIGCONT' : 'SIGSTOP')
  }

  async makeBackup(): Promise<void> {
    await this.requireOwn('make a backup of the whole server').makeBackup()
  }

  async restoreFromBackup(isTold: boolean): Promise<void> {
    await this.requireOwn('restore the whole server from a backup').restoreBackup(isTold)
  }

  async pause(): Promise<void> {
    await this.requireOwn('stop the server').stopServer()
  }

  async resume(): Promise<void> {
    const own = this.requireOwn('run the server again')
    await own.startServer()
    if (this.held.length > 0) {
      await new Promise((resolve) => setTimeout(resolve, HOLD_MS))
      await this.releaseHeld()
    }
  }

  async breakAfter(direction: 'received' | 'sent', bytes: number): Promise<void> {
    await this.requireOwn('break the connection of the server after some bytes').breakAfter(direction, bytes)
  }

  async forgetRemovals(): Promise<void> {
    await this.requireOwn('make the server drop the removals it keeps').forgetRemovals()
  }

  needNewerProtocol(): Promise<never> {
    return Promise.reject(this.createPowerError('make the server need a newer protocol'))
  }

  async dropStreams(): Promise<void> {
    await this.requireOwn('end every stream of the server').restartServer()
  }

  async endSession(): Promise<void> {
    await this.requireOwn('end the sessions of the server').endSessions()
  }
}
