/** Configures the sync server for a scenario and drives the server and the sync processes in that scenario. */
import { execFile } from 'node:child_process'
import { setTimeout as wait } from 'node:timers/promises'
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
/** Delay before the deferred computer's sync backend resumes. */
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

/** Finds the plugin among the descendants of this computer's application process. */
export const findChildBackendPids = (processes: string, root: number, backend: string): number[] => {
  const children = new Map<number, number[]>()
  const programs = new Map<number, string>()
  for (const line of processes.split('\n')) {
    const found = /^\s*(\d+)\s+(\d+)\s+(.*)$/.exec(line)
    if (found === null) continue
    const pid = Number(found[1])
    const parent = Number(found[2])
    const siblings = children.get(parent) ?? []
    siblings.push(pid)
    children.set(parent, siblings)
    programs.set(pid, pid + ' ' + found[3])
  }
  const family = new Set<number>([root])
  const pending = [root]
  while (pending.length > 0) {
    const parent = pending.pop()
    if (parent === undefined) break
    for (const pid of children.get(parent) ?? []) {
      if (family.has(pid)) continue
      family.add(pid)
      pending.push(pid)
    }
  }
  family.delete(root)
  return findBackendPids([...family].map((pid) => programs.get(pid)).join('\n'), backend)
}

export class SyncDriver {
  private stack: SyncStack | undefined
  private startedAt = new Date()
  private own: OwnStack | undefined
  private deferredScenario: Scenario | undefined
  private readonly paused = new Map<Scenario, Set<number>>()
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

  /** Returns the stand and the computers to a running state and removes the seed files. */
  async remove(): Promise<void> {
    const tasks = await Promise.allSettled([
      this.own?.resetServer(),
      ...[...this.paused.keys()].map((scenario) => this.resumePlugin(scenario)),
    ])
    if (this.seeded !== undefined) {
      removeSeed(this.seeded.folder)
      this.seeded = undefined
    }
    const failures = tasks.flatMap((task) => (task.status === 'rejected' ? [task.reason as unknown] : []))
    if (failures.length > 0) throw new AggregateError(failures, 'the sync scenario cleanup failed')
  }

  private requireStack(): SyncStack {
    if (this.stack === undefined) throw new Error('no sync server named: begin with a sync server runs')
    return this.stack
  }

  private async readProcesses(): Promise<ProcessRow[]> {
    const head = await this.scenario.getServer().readStreamHead('PluginProcessService/WatchPluginProcesses', {})
    return (head['processes'] ?? []) as ProcessRow[]
  }

  private requireOwn(power: string): OwnStack {
    const project = process.env[ENV_OWN_STACK]
    if (project === undefined || project === '') throw new Error(power + ': set ' + ENV_OWN_STACK + ' to the stand project for @stand')
    this.own ??= new OwnStack(project)
    return this.own
  }

  private async findPluginPids(scenario: Scenario): Promise<number[]> {
    const root = scenario.getServer().getPid()
    if (root === undefined) return []
    const listed = (await run('ps', ['-A', '-o', 'pid=,ppid=,args='])).stdout
    return findChildBackendPids(listed, root, findPlugin('Numen Sync').backend)
  }

  private async pausePlugin(scenario: Scenario): Promise<void> {
    const running = await this.findPluginPids(scenario)
    if (running.length === 0) throw new Error('the computer has no running Numen Sync backend to pause')
    const paused = this.paused.get(scenario) ?? new Set<number>()
    this.paused.set(scenario, paused)
    for (const pid of running) {
      process.kill(pid, 'SIGSTOP')
      paused.add(pid)
    }
  }

  private async resumePlugin(scenario: Scenario): Promise<void> {
    const paused = this.paused.get(scenario)
    if (paused === undefined) return
    for (const pid of await this.findPluginPids(scenario)) {
      if (!paused.has(pid)) continue
      try {
        process.kill(pid, 'SIGCONT')
      } catch (error) {
        if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error
      }
    }
    this.paused.delete(scenario)
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
  async holdOther(other: Scenario): Promise<void> {
    this.requireOwn('cut the other computers off from the server')
    await this.pausePlugin(other)
    this.deferredScenario = other
  }

  async setReachable(scenario: Scenario, isReachable: boolean): Promise<void> {
    this.requireOwn('cut one computer off from the server')
    if (isReachable) await this.resumePlugin(scenario)
    else await this.pausePlugin(scenario)
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
    if (this.deferredScenario !== undefined) {
      await wait(HOLD_MS)
      await this.resumePlugin(this.deferredScenario)
      this.deferredScenario = undefined
    }
  }

  async breakAfter(direction: 'received' | 'sent', bytes: number): Promise<void> {
    await this.requireOwn('break the connection of the server after some bytes').breakAfter(direction, bytes)
  }

  async waitForInterruption(): Promise<void> {
    await this.own?.waitForInterruption()
  }

  async forgetRemovals(): Promise<void> {
    await this.requireOwn('make the server drop the removals it keeps').forgetRemovals()
  }

  async dropStreams(): Promise<void> {
    await this.requireOwn('end every stream of the server').restartServer()
  }

  async endSession(): Promise<void> {
    await this.requireOwn('end the sessions of the server').endSessions()
  }
}
