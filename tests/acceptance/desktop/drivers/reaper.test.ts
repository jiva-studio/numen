import { spawn } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { createReaperLine } from './reaper.js'

const SCRIPT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../scripts/reap.mjs')

const ended = (child: ReturnType<typeof spawn>): Promise<void> => new Promise((resolve) => child.once('close', () => resolve()))

describe('createReaperLine', () => {
  it('writes the verb, the kind and the value on one line', () => {
    expect(createReaperLine('hold', 'dir', '/tmp/a b')).toBe('hold dir /tmp/a b\n')
    expect(createReaperLine('free', 'pid', 12)).toBe('free pid 12\n')
  })

  it('refuses a value with a line break', () => {
    expect(() => createReaperLine('hold', 'dir', '/tmp/a\nb')).toThrow('no line break')
  })
})

describe('the reaper script', () => {
  const run = async (lines: string[]): Promise<void> => {
    const reaper = spawn(process.execPath, [SCRIPT], { stdio: ['pipe', 'ignore', 'ignore'] })
    const exit = ended(reaper)
    reaper.stdin.end(lines.join(''))
    await exit
  }

  it('kills the processes and removes the folders named in hold lines when its input ends', async () => {
    const folder = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-reaper-test-'))
    fs.writeFileSync(path.join(folder, 'file'), 'x')
    const child = spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], { stdio: 'ignore' })
    const exit = ended(child)
    await run([createReaperLine('hold', 'pid', child.pid ?? 0), createReaperLine('hold', 'dir', folder)])
    await exit
    expect(fs.existsSync(folder)).toBe(false)
    expect(child.signalCode).toBe('SIGKILL')
  })

  it('leaves alone the processes and folders that free lines name', async () => {
    const folder = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-reaper-test-'))
    const child = spawn(process.execPath, ['-e', 'process.stdin.resume()'], { stdio: ['pipe', 'ignore', 'ignore'] })
    await run([
      createReaperLine('hold', 'pid', child.pid ?? 0),
      createReaperLine('hold', 'dir', folder),
      createReaperLine('free', 'pid', child.pid ?? 0),
      createReaperLine('free', 'dir', folder),
    ])
    expect(fs.existsSync(folder)).toBe(true)
    expect(child.exitCode).toBeNull()
    expect(child.signalCode).toBeNull()
    child.kill()
    fs.rmSync(folder, { recursive: true, force: true })
  })
})
