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
  baseline,
  config,
  depcruise,
  layered,
  layers,
  modules,
  root,
  screened,
  screens,
  unscreened,
} from './modules.mjs'

/**
 * The rules a module's own folders answer to, and one file under them the
 * cruise has to have reached. A module answering to none reads as null.
 */
const boundary = (name) => {
  if (screened.has(name)) return { rules: screens, reads: screened.get(name) }
  if (layered.has(name)) return { rules: layers, reads: layered.get(name) }
  return null
}

let broke = false

const wrong = (said) => {
  console.error(`  ${said}`)
  broke = true
}

// A module no boundary rule reads and that is not told to leave alone is a rule
// stopping at a border with nobody told, which is what the rule itself refuses
// one level down. Silence is the failure; a stated reason is not.
for (const { name } of modules) {
  const read = boundary(name) !== null
  const excused = Object.hasOwn(unscreened, name)
  if (read === excused) {
    console.log(name)
    wrong(
      read
        ? 'a boundary rule reads its folders and `unscreened` says why none does'
        : 'no boundary rule reads its folders and `unscreened` gives no reason',
    )
  }
}

for (const { name, at, sources, reads } of modules) {
  // A module whose own folders answer to a rule is read against that as well;
  // every other module against the rules every module answers to.
  const held = boundary(name)
  const rules = held ? held.rules : config
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

  const accepted = baseline.get(name) ?? []
  const standing = violations.map((one) => `${one.rule.name}: ${one.from} → ${one.to}`)
  for (const [at, one] of standing.entries()) {
    if (accepted.includes(one)) continue
    wrong(`${violations[at].rule.severity} ${one}`)
  }

  // An entry naming an edge nobody draws any more is a rule kept alive by a
  // line nobody reads. The list only shrinks.
  for (const one of accepted) {
    if (!standing.includes(one)) wrong(`in the baseline, and nobody draws it: ${one}`)
  }

  const read = new Set(cruised.modules.map((one) => one.source))
  if (!read.has(reads)) {
    wrong(`the cruise did not read ${reads}, so it walked a tree that is not this module's`)
  }
  // A boundary rule judges what stands under a folder. A cruise that reached a
  // module's root and no further would find nothing to judge and pass, which
  // reads exactly like a module whose folders are apart.
  if (held && !read.has(held.reads)) {
    wrong(`the cruise did not read ${held.reads}, so the boundary rule judged nothing`)
  }
}

process.exit(broke ? 1 : 0)
