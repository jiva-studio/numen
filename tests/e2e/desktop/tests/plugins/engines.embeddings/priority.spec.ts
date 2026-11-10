import { execFileSync } from 'node:child_process'
import { test, expect } from '../../support/test.js'
import { readOwner, writeSetting } from '../../support/vault.js'
import type { JsonObject, JsonValue } from '../../support/json.js'

interface Row {
  pid: number
  ppid: number
  nice: number
  args: string
}

const readProcesses = (): Row[] =>
  execFileSync('ps', ['-eo', 'pid=,ppid=,ni=,args='], { encoding: 'utf8' })
    .split('\n')
    .flatMap((line) => {
      const match = /^\s*(\d+)\s+(\d+)\s+(-?\d+)\s+(.*)$/.exec(line)
      return match
        ? [
            {
              pid: Number(match[1]),
              ppid: Number(match[2]),
              nice: Number(match[3]),
              args: match[4]!,
            },
          ]
        : []
    })

const readPluginNice = (): number | undefined => {
  const port = process.env['NUMEN_TEST_PORT'] ?? '3456'
  const rows = readProcesses()
  const address = (arg: string) => arg.startsWith('-debug-http-addr=') && arg.endsWith(`:${port}`)
  const server = rows.find((row) => row.args.split(' ').some(address))
  if (!server) return undefined
  const owned = new Set([server.pid])
  for (let grew = true; grew;) {
    grew = false
    for (const row of rows) {
      if (owned.has(row.ppid) && !owned.has(row.pid)) {
        owned.add(row.pid)
        grew = true
      }
    }
  }
  return rows.find((row) => owned.has(row.pid) && /engines-embeddings\b/.test(row.args))?.nice
}

test.describe('Performance profile of the embeddings plugin', () => {
  let profile: JsonValue | undefined

  test.beforeEach(async ({ request }) => {
    profile = (
      (await readOwner(request, 'app'))['performance'] as JsonObject | undefined
    )?.['profile']
  })

  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'performance', 'profile'], profile)
  })

  test('lowers the priority of the running plugin when the profile is eco', async ({ request }) => {
    await writeSetting(request, ['app', 'performance', 'profile'], 'eco')

    await expect.poll(readPluginNice, { timeout: 15_000 }).toBe(19)
  })
})
