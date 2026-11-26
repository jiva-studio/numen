/**
 * The tests a plugin folder holds, counted from its source without running them: Go test and fuzz
 * functions in `*_test.go`, TypeScript test cases in `*.test.ts` and stories in `*.stories.ts`,
 * which run as browser tests; and the Vue components no story shows.
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, join, relative, resolve, sep } from 'node:path'

const SKIPPED_FOLDERS = new Set(['node_modules', 'dist', 'testdata', '.git'])

const GO_TEST = /^func (?:Test|Fuzz)\w*\(\s*\w+\s+\*testing\.[TF]\s*\)/gm

const TS_TEST =
  /(?<![\w.$])(?:it|test)(?:\.(?:each|only|skip|todo|concurrent|fails|sequential))*\(/g

const STORY = /^export const [A-Z]\w*/gm

const COUNTERS = [
  { suffix: '_test.go', kind: 'go', pattern: GO_TEST },
  { suffix: '.test.ts', kind: 'typescript', pattern: TS_TEST },
  { suffix: '.stories.ts', kind: 'stories', pattern: STORY },
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
  const counts = { go: 0, typescript: 0, stories: 0 }
  for (const file of listFiles(dir)) {
    const counter = COUNTERS.find((one) => file.endsWith(one.suffix))
    if (!counter) continue
    counts[counter.kind] += readFileSync(file, 'utf8').match(counter.pattern)?.length ?? 0
  }
  return counts
}

const VUE_IMPORT = /(?:from\s+|import\s*\(\s*)["'](\.{1,2}\/[^"']+\.vue)["']/g

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
        [...readFileSync(file, 'utf8').matchAll(VUE_IMPORT)].map((match) =>
          resolve(dirname(file), match[1]),
        ),
      ),
  )
  const components = files.filter((file) => file.startsWith(frontend + sep) && file.endsWith('.vue')).sort()
  const toName = (file) => relative(dir, file).split(sep).join('/')
  return {
    components: components.map(toName),
    unstoried: components.filter((file) => !storied.has(file)).map(toName),
  }
}
