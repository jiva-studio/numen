// Stays beside a worker of the acceptance suite and cleans up after it. The worker writes a line
// for each process and each folder it makes ("hold pid 12", "hold dir /tmp/x"), and a line when it
// has ended or removed that itself ("free pid 12"). When the worker's end of the pipe closes,
// however the worker ended (a kill included), the processes and folders still on the registry are ended and removed.
import fs from 'node:fs'
import readline from 'node:readline'

const registry = { pid: new Set(), dir: new Set() }

const lines = readline.createInterface({ input: process.stdin })
lines.on('line', (line) => {
  const [verb, kind, value] = line.split(' ', 3)
  const rest = line.slice(verb.length + kind.length + 2)
  if (!(kind in registry)) return
  if (verb === 'hold') registry[kind].add(kind === 'pid' ? Number(value) : rest)
  if (verb === 'free') registry[kind].delete(kind === 'pid' ? Number(value) : rest)
})
lines.on('close', () => {
  for (const pid of registry.pid) {
    try {
      process.kill(pid, 'SIGKILL')
    } catch (error) {
      // A process that is gone already needs no kill.
      if (error.code !== 'ESRCH') throw error
    }
  }
  for (const dir of registry.dir) fs.rmSync(dir, { recursive: true, force: true })
})
