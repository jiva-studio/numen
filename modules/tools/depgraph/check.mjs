/**
 * Every module cruised, and a run that fails if any of them broke a rule.
 *
 * A cruise that read the wrong tree finds nothing and says so in the same words
 * as a cruise that read the right one, so each module names one file its walk
 * has to have reached. A count on its own cannot tell the two apart.
 */
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import { config, depcruise, modules, root } from './modules.mjs'

let broke = false

const wrong = (said) => {
  console.error(`  ${said}`)
  broke = true
}

for (const { name, at, sources, reads } of modules) {
  const run = spawnSync(depcruise, ['--config', config, '--output-type', 'json', ...sources], {
    cwd: join(root, at),
    encoding: 'utf8',
    maxBuffer: 256 * 1024 * 1024,
  })
  if (run.error || run.stdout === '') {
    console.log(`${name} (${at})`)
    wrong(`the cruise did not run: ${run.stderr?.trim() || run.error}`)
    continue
  }

  const cruised = JSON.parse(run.stdout)
  const { violations, totalCruised, totalDependenciesCruised } = cruised.summary
  console.log(`${name} (${at}): ${totalCruised} modules, ${totalDependenciesCruised} dependencies`)

  for (const one of violations) {
    wrong(`${one.rule.severity} ${one.rule.name}: ${one.from} → ${one.to}`)
  }
  if (!cruised.modules.some((one) => one.source === reads)) {
    wrong(`the cruise did not read ${reads}, so it walked a tree that is not this module's`)
  }
}

process.exit(broke ? 1 : 0)
