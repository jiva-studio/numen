import assert from 'node:assert/strict'
import { execFileSync, spawnSync } from 'node:child_process'
import {
  cpSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { after, test } from 'node:test'
import { fileURLToPath } from 'node:url'

const root = resolve(fileURLToPath(import.meta.url), '../../..')
const folder = 'libs/core/adapter/index/migration'
const script = resolve(root, 'libs/core/adapter/index/migration-forward-only.sh')
// The ten files of the sequence before the collapse.
const tenSteps = resolve(root, '.github/scripts/fixtures/index-ten-steps')
const rooms = []

after(() => {
  for (const room of rooms) rmSync(room, { recursive: true, force: true })
})

const git = (cwd, ...args) =>
  execFileSync('git', args, {
    cwd,
    encoding: 'utf8',
    env: { ...process.env, GIT_DIR: undefined },
  }).trim()

const record = (name) => {
  const found = readFileSync(script, 'utf8').match(new RegExp(`^${name}=(\\S+)$`, 'm'))
  assert.ok(found, `the script holds no record ${name}`)
  return found[1]
}

// A repository whose main carries the ten old files and whose branch carries the folder as it stands.
function build() {
  const room = mkdtempSync(join(tmpdir(), 'forward-only-'))
  rooms.push(room)
  git(room, 'init', '-q', '-b', 'main', '.')
  git(room, 'config', 'user.email', 'check@numen.invalid')
  git(room, 'config', 'user.name', 'check')
  mkdirSync(join(room, folder), { recursive: true })
  cpSync(tenSteps, join(room, folder), { recursive: true })
  git(room, 'add', '--', folder)
  git(room, 'commit', '-qm', 'base')
  git(room, 'switch', '-qc', 'branch')
  rmSync(join(room, folder), { recursive: true })
  cpSync(resolve(root, folder), join(room, folder), { recursive: true })
  git(room, 'add', '-A', '--', folder)
  git(room, 'commit', '-q', '--allow-empty', '-m', 'collapse')
  return room
}

const check = (room) =>
  spawnSync(script, [git(room, 'rev-parse', 'main')], { cwd: room, encoding: 'utf8' })

test('the record of the forward-only rule names the ten old files and the one file that is left', () => {
  const room = build()
  assert.equal(record('collapsed'), git(room, 'rev-parse', `main:${folder}`))
  const left = readdirSync(resolve(root, folder))
  assert.deepEqual(left, [record('kept')])
  assert.equal(record('produced'), git(room, 'hash-object', join(room, folder, left[0])))
})

test('the check against a base that carries the ten old files lets the collapse through', () => {
  const run = check(build())
  assert.equal(run.status, 0, run.stderr)
})

test('the check refuses a rewrite of the one file that is left', () => {
  const room = build()
  const files = readdirSync(resolve(root, folder))
  assert.equal(files.length, 1, `the folder holds ${files.length} files, want the one that is left`)
  const [left] = files
  writeFileSync(join(room, folder, left), '\nCREATE TABLE rewritten (id INTEGER);\n', { flag: 'a' })
  git(room, 'commit', '-qam', 'rewrite')
  const run = check(room)
  assert.equal(run.status, 1, run.stderr)
  assert.match(run.stderr, new RegExp(left))
})

test('the self-test of the rule passes', () => {
  const run = spawnSync(script, ['--self-test'], { cwd: root, encoding: 'utf8' })
  assert.equal(run.status, 0, run.stderr)
})
