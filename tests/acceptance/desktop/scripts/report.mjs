import { spawnSync } from 'node:child_process'
import fs from 'node:fs'

const run = (args, env = process.env) =>
  spawnSync('npm', ['run', '-s', ...args], { stdio: 'inherit', env }).status ?? 1

fs.rmSync('acceptance-blobs', { recursive: true, force: true })
const tested = run(['acceptance:test', '--', ...process.argv.slice(2)], {
  ...process.env,
  PLAYWRIGHT_VIDEO: '1',
  PWTEST_BLOB_DO_NOT_REMOVE: '1',
})
const merged = run(['acceptance:merge'])
process.exit(tested || merged)
