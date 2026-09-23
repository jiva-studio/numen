/**
 * Every story the tree declares, under the name Storybook addresses it by, and
 * what is wrong with a page's shots held against them.
 *
 * A camera names its stories as strings, and a string is not a reference:
 * renaming a story leaves the name pointing at nothing, and nothing says so
 * until somebody runs the camera — which for a released page happens months
 * later, and shows up as a picture that never arrived.
 *
 * Two pages hold their shots to this, and a copy each is two rules that agree
 * only until one of them is touched.
 */
import { readdir, readFile } from 'node:fs/promises'
import { join } from 'node:path'
import { modules, root } from '../modules.mjs'

/**
 * Everywhere a story can be written: the hand-written source of every module
 * the repository has. Storybook is pointed at three of these folders today,
 * and a list here would be a fourth copy of the same three.
 */
const ROOTS = modules.map((one) => join(root, one.written))

/**
 * Every story id the tree declares, spelled as Storybook spells it: the meta's
 * own title and the export's name, each cut to lower case and dashes.
 */
export const stories = async () => {
  const found = new Set()
  for (const at of ROOTS) {
    const files = (await readdir(at, { recursive: true })).filter((name) =>
      name.endsWith('.stories.ts'),
    )
    for (const file of files) {
      const source = await readFile(join(at, file), 'utf8')
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

/**
 * What is wrong with these shots, held against these stories.
 *
 * Nothing to hold the names against is nothing checked, and so is no name: a
 * camera pointed at an empty tree passes every shot it is given, which is the
 * one way this rule can be true and worthless at once.
 */
export const faults = (shots, declared) => {
  const wrong = []
  if (declared.size === 0) wrong.push('no stories are written anywhere the camera is pointed')
  if (shots.length === 0) wrong.push('shoot.mjs asks for no pictures')
  const gone = shots.filter((shot) => !declared.has(shot.story))
  if (gone.length > 0) {
    wrong.push(
      `no story called ${gone.map((shot) => shot.story).join(', ')} — the pictures cannot be taken again`,
    )
  }
  return wrong
}

/* ------------------------------------------------- what the rule refuses */

/**
 * The rule put through lists made up here, each case with the words the rule
 * is to answer it in. A check nobody has seen fail is an assumption, and this
 * one's whole worth is its failure path, so the path is walked on every run
 * before a picture is.
 *
 * The words and not a yes or no: three of the rules below refuse the same
 * lists, so a case asking only whether something was refused holds two of them
 * up on the third and would let either be deleted.
 */
export const REFUSES = [
  {
    what: 'a shot naming a story that is declared',
    shots: [{ story: 'application-window--filing' }],
    declared: ['application-window--filing'],
    said: [],
  },
  {
    what: 'a shot naming a story that was renamed away',
    shots: [{ story: 'flash-cards-window--owing' }],
    declared: ['flash-cards-window--cards-due'],
    said: ['no story called flash-cards-window--owing — the pictures cannot be taken again'],
  },
  {
    what: 'one shot of several naming nothing',
    shots: [{ story: 'application-window--filing' }, { story: 'application-window--gone' }],
    declared: ['application-window--filing'],
    said: ['no story called application-window--gone — the pictures cannot be taken again'],
  },
  {
    what: 'a tree in which no story is written at all',
    shots: [{ story: 'application-window--filing' }],
    declared: [],
    said: [
      'no stories are written anywhere the camera is pointed',
      'no story called application-window--filing — the pictures cannot be taken again',
    ],
  },
  {
    what: 'a camera asking for no pictures',
    shots: [],
    declared: ['application-window--filing'],
    said: ['shoot.mjs asks for no pictures'],
  },
]

/** What the table caught, or null. The caller is the one that dies. */
export const proves = () => {
  if (REFUSES.length === 0) return 'the rule is put through nothing'
  for (const one of REFUSES) {
    const said = faults(one.shots, new Set(one.declared))
    if (said.length !== one.said.length || said.some((was, at) => was !== one.said[at])) {
      return `of ${one.what} the rule says ${JSON.stringify(said)}, and not ${JSON.stringify(one.said)}`
    }
  }
  return null
}
