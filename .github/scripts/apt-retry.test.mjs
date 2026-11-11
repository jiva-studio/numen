// apt-retry.sh over commands made up here: one that fails a given number of times and then
// succeeds, and one that never succeeds. The refresh command only records that it ran.
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterEach, beforeEach, describe, it } from 'node:test'

const SCRIPT = new URL('./apt-retry.sh', import.meta.url).pathname

let dir

beforeEach(() => {
  dir = mkdtempSync(join(tmpdir(), 'apt-retry-'))
  writeFileSync(
    join(dir, 'flaky.sh'),
    [
      'n=$(cat "$1/calls" 2>/dev/null || echo 0); n=$((n + 1)); echo $n > "$1/calls"',
      'if [ "$n" -lt "$2" ]; then echo "E: Temporary failure resolving (attempt $n)"; exit 100; fi',
      'echo "installed on attempt $n"',
      '',
    ].join('\n'),
  )
  writeFileSync(join(dir, 'refresh.sh'), `echo refresh >> "${dir}/refreshes"\n`)
})

afterEach(() => rmSync(dir, { recursive: true, force: true }))

const run = (args) =>
  spawnSync('bash', [SCRIPT, ...args], {
    encoding: 'utf8',
    env: {
      ...process.env,
      APT_RETRY_DELAYS: '0 0 0',
      APT_RETRY_CONF: join(dir, 'apt.conf'),
      APT_RETRY_UPDATE: `bash ${join(dir, 'refresh.sh')}`,
    },
  })

const count = (file) => {
  try {
    return readFileSync(join(dir, file), 'utf8').trim().split('\n').length
  } catch {
    return 0
  }
}

const flaky = (succeedAt) => ['bash', join(dir, 'flaky.sh'), dir, String(succeedAt)]

describe('apt-retry.sh', () => {
  it('runs the command once when it succeeds and refreshes nothing', () => {
    const result = run(flaky(1))
    assert.equal(result.status, 0)
    assert.equal(readFileSync(join(dir, 'calls'), 'utf8').trim(), '1')
    assert.equal(count('refreshes'), 0)
  })

  it('succeeds on the third attempt, refreshing before each retry', () => {
    const result = run(flaky(3))
    assert.equal(result.status, 0)
    assert.equal(readFileSync(join(dir, 'calls'), 'utf8').trim(), '3')
    assert.equal(count('refreshes'), 2)
    assert.match(result.stdout, /installed on attempt 3/)
  })

  it('fails after the fourth attempt with the exit code and the output', () => {
    const result = run(flaky(99))
    assert.equal(result.status, 100)
    assert.equal(readFileSync(join(dir, 'calls'), 'utf8').trim(), '4')
    assert.equal(count('refreshes'), 3)
    assert.match(result.stdout, /::error::apt failed after 4 attempts \(exit 100\)/)
    assert.match(result.stdout, /Output of the last attempt:\nE: Temporary failure resolving \(attempt 4\)/)
  })

  it('refreshes before the first attempt with --update', () => {
    const result = run(['--update', ...flaky(1)])
    assert.equal(result.status, 0)
    assert.equal(count('refreshes'), 1)
  })

  it('writes the apt options for the commands that call apt themselves', () => {
    run(flaky(1))
    const conf = readFileSync(join(dir, 'apt.conf'), 'utf8')
    assert.match(conf, /Acquire::Retries "5"/)
    assert.match(conf, /Acquire::http::Timeout "30"/)
    assert.match(conf, /Acquire::https::Timeout "30"/)
  })
})
