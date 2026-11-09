// Stays beside a worker of the acceptance suite and cleans up after it. The worker writes a line
// for each process and each folder it makes ("hold pid 12", "hold dir /tmp/x"), and a line when it
// has ended or removed that itself ("free pid 12"). When the worker's end of the pipe closes,
// however the worker ended (a kill included), the processes and folders still held are ended and removed.
import fs from 'node:fs'
import readline from 'node:readline'

const held = { pid: new Set(), dir: new Set() }

const lines = readline.createInterface({ input: process.stdin })
lines.on('line', (line) => {
  const [verb, kind, value] = line.split(' ', 3)
  const rest = line.slice(verb.length + kind.length + 2)
  if (!(kind in held)) return
  if (verb === 'hold') held[kind].add(kind === 'pid' ? Number(value) : rest)
  if (verb === 'free') held[kind].delete(kind === 'pid' ? Number(value) : rest)
})
lines.on('close', () => {
  for (const pid of held.pid) {
    try {
      process.kill(pid, 'SIGKILL')
    } catch {
      // The process is gone already.
    }
  }
  for (const dir of held.dir) fs.rmSync(dir, { recursive: true, force: true })
})
