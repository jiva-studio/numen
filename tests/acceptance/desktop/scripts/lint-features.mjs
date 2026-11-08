// Runs gplint on every .feature file of `features/`, by explicit path. gplint expands a glob that starts
// with `../` to nothing and exits 0, and it exits 0 when its own configuration is wrong, so a run
// that read no file, or that printed its own error, fails here.
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const features = path.resolve(root, '../../../features')
const files = fs
  .readdirSync(features, { recursive: true })
  .map(String)
  .filter((name) => name.endsWith('.feature'))
  .map((name) => path.join(features, name))

if (files.length === 0) {
  console.error('lint-features: no .feature file under features/')
  process.exit(1)
}

const result = spawnSync('npx', ['gplint', '-c', path.join(root, '.gplintrc'), ...files], {
  cwd: root,
  encoding: 'utf8',
})
process.stdout.write(result.stdout)
process.stderr.write(result.stderr)
if (result.status !== 0 || /Error running gplint/.test(result.stderr + result.stdout)) process.exit(1)
console.log(`lint-features: ${files.length} feature files`)
