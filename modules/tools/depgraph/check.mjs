/** Every module cruised, and a run that fails if any of them broke a rule. */
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import { config, depcruise, modules, root } from './modules.mjs'

let broke = false

for (const { name, at, sources } of modules) {
  console.log(`${name} (${at})`)
  const run = spawnSync(depcruise, ['--config', config, '--output-type', 'err', ...sources], {
    cwd: join(root, at),
    stdio: 'inherit',
  })
  if (run.status !== 0) broke = true
}

process.exit(broke ? 1 : 0)
