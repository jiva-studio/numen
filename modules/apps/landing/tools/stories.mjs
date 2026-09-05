/**
 * That every picture this page shows is still a story somebody can take again.
 *
 * The camera names its stories as strings, and a string is not a reference:
 * renaming a story leaves the name here pointing at nothing, and nothing says
 * so until somebody runs the camera — which for this page happens at a
 * release, months later, and shows up as a picture that never arrived. So the
 * names are held against the tree here, by a check that runs with the rest.
 */
import { readdir, readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'

import { SHOTS } from './shots.mjs'

/** Everywhere a story is written, which is the same list Storybook is given. */
const ROOTS = [
  new URL('../../../libs/ui/src/', import.meta.url),
  new URL('../../desktop/ui/src/', import.meta.url),
  new URL('../../desktop/flashcards/src/', import.meta.url),
]

const die = (message) => {
  process.stderr.write(`stories: ${message}\n`)
  process.exit(1)
}

/** Every story id the tree declares, under the same name Storybook gives it. */
const told = async () => {
  const found = new Set()
  for (const root of ROOTS) {
    const files = (await readdir(root, { recursive: true })).filter((name) =>
      name.endsWith('.stories.ts'),
    )
    for (const file of files) {
      const source = await readFile(new URL(file, root), 'utf8')
      const meta = source.match(/const meta[^=]*=\s*\{[\s\S]{0,200}?title:\s*'([^']+)'/)
      if (!meta) continue
      const under = meta[1].toLowerCase().replace(/[^a-z0-9]+/g, '-')
      for (const [, name] of source.matchAll(/^export const ([A-Z][A-Za-z0-9]*)\s*:/gm)) {
        found.add(`${under}--${name.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase()}`)
      }
    }
  }
  return found
}

/** Which shots name a story nothing declares. The one rule this file holds. */
const gone = (shots, declared) => shots.filter((shot) => !declared.has(shot.story))

/* ------------------------------------------------- what the rule refuses */

/**
 * The rule put through lists made up here. A check nobody has seen fail is an
 * assumption, and this one's whole worth is its failure path.
 */
const REFUSES = [
  {
    what: 'a shot naming a story that is declared',
    shots: [{ story: 'application-window--filing' }],
    declared: ['application-window--filing'],
    refused: false,
  },
  {
    what: 'a shot naming a story that was renamed away',
    shots: [{ story: 'flash-cards-window--owing' }],
    declared: ['flash-cards-window--cards-due'],
    refused: true,
  },
  {
    what: 'one shot of several naming nothing',
    shots: [{ story: 'application-window--filing' }, { story: 'application-window--gone' }],
    declared: ['application-window--filing'],
    refused: true,
  },
]

const refuses = () => {
  if (REFUSES.length === 0) die('the rule is put through nothing')
  for (const one of REFUSES) {
    const refused = gone(one.shots, new Set(one.declared)).length > 0
    if (refused !== one.refused) {
      die(`the rule ${one.refused ? 'lets through' : 'refuses'} ${one.what}`)
    }
  }
}

/* ------------------------------------------------------------------ ask */

export const checked = async () => {
  refuses()

  const declared = await told()
  // Nothing to hold the names against is nothing checked, and so is no name.
  if (declared.size === 0) die('no stories are written anywhere the camera is pointed')
  if (SHOTS.length === 0) die('shoot.mjs asks for no pictures')

  const missing = gone(SHOTS, declared)
  if (missing.length > 0) {
    die(`no story called ${missing.map((shot) => shot.story).join(', ')} — the pictures cannot be taken again`)
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  await checked()
  process.stdout.write(`stories: every picture on the page is still a story\n`)
}
