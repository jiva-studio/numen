/** The helper process that kills the servers and removes the folders of a worker that ended without cleaning up. */
import { spawn, type ChildProcess } from 'node:child_process'
import path from 'node:path'
import type { Writable } from 'node:stream'
import { fileURLToPath } from 'node:url'

const SCRIPT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../scripts/reap.mjs')

export type Held = 'pid' | 'dir'

/** The line the helper reads to start or stop holding a process or a folder. */
export const createReaperLine = (verb: 'hold' | 'free', kind: Held, value: string | number): string => {
  const text = String(value)
  if (text.includes('\n')) throw new Error('a process or a folder held for removal has no line break in its name')
  return `${verb} ${kind} ${text}\n`
}

let helper: ChildProcess | undefined

const getHelper = (): ChildProcess => {
  if (helper?.stdin?.writable) return helper
  const child = spawn(process.execPath, [SCRIPT], { stdio: ['pipe', 'ignore', 'ignore'], detached: true })
  child.unref()
  const stdin = child.stdin as (Writable & { unref?: () => void }) | null
  stdin?.unref?.()
  stdin?.on('error', () => undefined)
  helper = child
  return child
}

/** Has the helper end the process, or remove the folder, when this process ends and the caller has not freed it. */
export const holdForRemoval = (kind: Held, value: string | number): void => {
  getHelper().stdin?.write(createReaperLine('hold', kind, value))
}

/** Tells the helper that the process has ended, or the folder is removed. */
export const freeFromRemoval = (kind: Held, value: string | number): void => {
  getHelper().stdin?.write(createReaperLine('free', kind, value))
}
