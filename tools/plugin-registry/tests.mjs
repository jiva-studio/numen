/**
 * The tests a plugin folder holds, counted from the syntax trees of its source without running
 * them: Go test and fuzz functions in `*_test.go`, TypeScript test cases in `*.test.ts` and stories
 * in `*.stories.ts`, which run as browser tests; and the Vue components no story shows.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, join, relative, resolve, sep } from 'node:path'
import { readGoFiles } from './gosource.mjs'
import { countStories, countTestCases, listImportedModules, parseTypeScript } from './tssource.mjs'

const SKIPPED_FOLDERS = new Set(['node_modules', 'dist', 'testdata', '.git'])

const TS_COUNTERS = [
  { suffix: '.test.ts', kind: 'typescript', count: countTestCases },
  { suffix: '.stories.ts', kind: 'stories', count: countStories },
]

const listFiles = (dir) =>
  readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    if (entry.isDirectory()) {
      return SKIPPED_FOLDERS.has(entry.name) ? [] : listFiles(join(dir, entry.name))
    }
    return entry.isFile() ? [join(dir, entry.name)] : []
  })

/**
 * The tests of the folder `dir`, by kind: Go functions, TypeScript cases (an `it.each` table
 * counts once) and stories.
 *
 * @param {string} dir
 * @returns {{ go: number, typescript: number, stories: number }}
 */
export const countTests = (dir) => {
  const files = listFiles(dir)
  const counts = { go: 0, typescript: 0, stories: 0 }
  for (const facts of readGoFiles(files.filter((file) => file.endsWith('_test.go'))).values()) {
    counts.go += facts.tests
  }
  for (const file of files) {
    const counter = TS_COUNTERS.find((one) => file.endsWith(one.suffix))
    if (counter) counts[counter.kind] += counter.count(parseTypeScript(readFileSync(file, 'utf8')))
  }
  return counts
}

const isRelativeVueModule = (module) =>
  (module.startsWith('./') || module.startsWith('../')) && module.endsWith('.vue')

/**
 * The Vue components of the folder `frontend` of `dir` and the ones no story shows: a component
 * has a story when a `*.stories.ts` of the plugin imports its file by a relative path. Null when
 * the plugin has no `frontend` folder.
 *
 * @param {string} dir - the plugin folder
 * @returns {{ components: string[], unstoried: string[] } | null} paths relative to `dir`
 */
export const listUnstoriedComponents = (dir) => {
  const frontend = join(dir, 'frontend')
  if (!existsSync(frontend)) return null
  const files = listFiles(dir)
  const storied = new Set(
    files
      .filter((file) => file.endsWith('.stories.ts'))
      .flatMap((file) =>
        listImportedModules(parseTypeScript(readFileSync(file, 'utf8')))
          .filter(isRelativeVueModule)
          .map((module) => resolve(dirname(file), module)),
      ),
  )
  const components = files.filter((file) => file.startsWith(frontend + sep) && file.endsWith('.vue')).sort()
  const toName = (file) => relative(dir, file).split(sep).join('/')
  return {
    components: components.map(toName),
    unstoried: components.filter((file) => !storied.has(file)).map(toName),
  }
}
