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
import { modules, root } from '../shared/modules.mjs'

/**
 * The two Storybooks a story is drawn in, each by the folder it is started
 * from: the components' own, which draws the windows' screens beside them, and
 * the plugins', which draws every plugin's frontend.
 */
export const BOOKS = {
  ui: 'libs/ui/core',
  plugins: 'plugins',
}

/**
 * Everywhere a story can be written, and the Storybook that draws it: the
 * hand-written source of every module the repository has, and the frontend of
 * every plugin.
 */
const ROOTS = [
  ...modules.map((one) => ({ at: join(root, one.written), book: 'ui', only: /./ })),
  { at: join(root, BOOKS.plugins), book: 'plugins', only: /^[^/]+\/frontend\// },
]

/**
 * Every story id the tree declares, spelled as Storybook spells it — the meta's
 * own title and the export's name, each cut to lower case and dashes — against
 * the Storybook it is drawn in. An id two Storybooks declare is drawn in
 * `both`, which names neither.
 */
export const stories = async () => {
  const found = new Map()
  for (const { at, book, only } of ROOTS) {
    const files = (await readdir(at, { recursive: true })).filter(
      (name) => name.endsWith('.stories.ts') && only.test(name) && !name.includes('node_modules'),
    )
    for (const file of files) {
      const source = await readFile(join(at, file), 'utf8')
      const meta = source.match(/const meta[^=]*=\s*\{[\s\S]{0,200}?title:\s*'([^']+)'/)
      if (!meta) continue
      const under = meta[1].toLowerCase().replace(/[^a-z0-9]+/g, '-')
      for (const [, name] of source.matchAll(/^export const ([A-Z][A-Za-z0-9]*)\s*:/gm)) {
        const words = name
          .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
          .replace(/([A-Z])([A-Z][a-z])/g, '$1-$2')
        const id = `${under}--${words.toLowerCase()}`
        const before = found.get(id)
        found.set(id, before && before !== book ? 'both' : book)
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
  const twice = shots.filter((shot) => declared.get(shot.story) === 'both')
  if (twice.length > 0) {
    wrong.push(
      `both Storybooks declare ${twice.map((shot) => shot.story).join(', ')} — the camera cannot tell which to start`,
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
    what: 'a shot naming a story both Storybooks declare',
    shots: [{ story: 'flash-cards-window--cards-due' }],
    declared: [['flash-cards-window--cards-due', 'both']],
    said: [
      'both Storybooks declare flash-cards-window--cards-due — the camera cannot tell which to start',
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
    const declared = new Map(
      one.declared.map((entry) => (Array.isArray(entry) ? entry : [entry, 'ui'])),
    )
    const said = faults(one.shots, declared)
    if (said.length !== one.said.length || said.some((was, at) => was !== one.said[at])) {
      return `of ${one.what} the rule says ${JSON.stringify(said)}, and not ${JSON.stringify(one.said)}`
    }
  }
  return null
}
