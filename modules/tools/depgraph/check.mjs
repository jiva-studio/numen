/**
 * Every module cruised, and a run that fails if any of them broke a rule.
 *
 * A cruise that read the wrong tree finds nothing and says so in the same words
 * as a cruise that read the right one, so each module names one file its walk
 * has to have reached. A count on its own cannot tell the two apart.
 */
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import {
  config,
  depcruise,
  modules,
  owed,
  root,
  screened,
  screens,
  unscreened,
} from './modules.mjs'

let broke = false

const wrong = (said) => {
  console.error(`  ${said}`)
  broke = true
}

// A module the screen rule neither reads nor is told to leave alone is a rule
// stopping at a border with nobody told, which is what the rule itself refuses
// one level down. Silence is the failure; a stated reason is not.
for (const { name } of modules) {
  const read = screened.has(name)
  const excused = Object.hasOwn(unscreened, name)
  if (read === excused) {
    console.log(name)
    wrong(
      read
        ? 'the screen rule reads it and `unscreened` says why it does not'
        : 'the screen rule does not read it and `unscreened` gives no reason',
    )
  }
}

for (const { name, at, sources, reads } of modules) {
  // A window is read against the screen rule as well; every other module
  // against the rules every module answers to.
  const screen = screened.get(name)
  const rules = screen ? screens : config
  const run = spawnSync(depcruise, ['--config', rules, '--output-type', 'json', ...sources], {
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
  const aside = unscreened[name] ? `, no screens: ${unscreened[name]}` : ''
  console.log(
    `${name} (${at}): ${totalCruised} modules, ${totalDependenciesCruised} dependencies${aside}`,
  )

  const debts = owed.get(name) ?? []
  const standing = violations.map((one) => `${one.rule.name}: ${one.from} → ${one.to}`)
  for (const [at, one] of standing.entries()) {
    if (debts.includes(one)) continue
    wrong(`${violations[at].rule.severity} ${one}`)
  }

  // An entry naming an edge nobody draws any more is a rule kept alive by a
  // line nobody reads. The list only shrinks.
  for (const one of debts) {
    if (!standing.includes(one)) wrong(`owed, and nobody draws it: ${one}`)
  }

  const read = new Set(cruised.modules.map((one) => one.source))
  if (!read.has(reads)) {
    wrong(`the cruise did not read ${reads}, so it walked a tree that is not this module's`)
  }
  // The screen rule judges what stands under a screen folder. A cruise that
  // reached a module's root and no further would find nothing to judge and
  // pass, which reads exactly like a window whose screens are apart.
  if (screen && !read.has(screen)) {
    wrong(`the cruise did not read ${screen}, so the screen rule judged no screen`)
  }
}

process.exit(broke ? 1 : 0)
