/**
 * Prints, as `name=value` lines for $GITHUB_OUTPUT, which areas of the repository a change touches
 * and so which CI jobs run.
 *
 * Usage: git diff --name-only <base> <head> | node changes.mjs <event> [-]
 *
 * With `-` the changed files are read from standard input, one per line; without it, or with an
 * empty input, every area runs. The plugins are the folders under plugins/ that hold a manifest.json.
 *
 * A pull request runs the areas its files touch and every area that depends on them. A push, a
 * manual run, a change to code every area reads (SHARED) and a base missing from the clone run
 * every area. `plugin_dirs` lists the plugin folders to lint, build and test. `docs_deploy` is true
 * on a push that changes what the manual is written from. The end-to-end tests a change needs are
 * tests/e2e/desktop/scripts/select-tests.mjs's to choose.
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs'

export const AREAS = [
  'core',
  'protocol',
  'desktop',
  'desktop_fe',
  'ui',
  'sdk_go',
  'mobile',
  'mobile_fe',
  'plugin_stories',
  'plugin_versions',
  'docs',
  'references',
  'code',
  'narrative',
  'go',
  'js',
  'workflows',
  'gradle',
  'staging',
  'mutation_tool',
]

const ALL_PLUGINS = 'plugins'

/** Read by every area: the workspace, the build, the shared tools and CI itself. */
const SHARED =
  /^(go\.work(\.sum)?|package(-lock)?\.json|Makefile|\.node-version|tsconfig\.base\.json|prettier\.config\.mjs|\.prettierignore|stylelint\.config\.mjs)$|^tools\/shared\/|^tools\/lint-layers\/base-eslint\.mjs$|^\.github\/actions\/|^\.github\/scripts\/changes|^\.github\/workflows\/(ci|changes)\.yml$/

/** The Go lint rules every Go module is held to. */
const GO_RULES = /^\.golangci(-structure)?\.yml$|(^|\/)gorules\//

const GO_LIBS = ['core', 'desktop', 'mobile', 'sdk_go', ALL_PLUGINS]

/** What the manual writes itself from, besides apps/docs. */
const MANUAL_SOURCES =
  /^apps\/desktop\/editor\/src\/(features\/command-palette\/lib\/|features\/command-palette\/model\/palette\/shortcuts\.ts$|shared\/words\.ts$|app\/commands\/fileMenuCommands\.ts$|app\/window\/useSettings\.ts$)|^apps\/desktop\/cmd\/numen\/main\.go$|^libs\/core\/(adapter\/cli\/cli\.go$|container\/(settings|indexing)\.go$|adapter\/settings\/|usecase\/completion\/)|^plugins\/[^/]+\/(manifest\.json$|frontend\/index\.ts$|frontend\/.*words\.ts$)|^tools\/ci-commands\//

/** Story names the manual and the landing page hold, and the trees they are held against. */
const STORY_SOURCES =
  /^libs\/ui\/(core|shell)\/src\/|^apps\/desktop\/(editor|flashcards)\/src\/|^plugins\/[^/]+\/frontend\/|^plugins\/themes-presets\/themes\/|^tools\/ci-(stories|commands)\/|^apps\/docs\/(tools|src\/content)\/|^apps\/landing\/tools\//

/** Each rule adds its areas to a file it matches; a file can match several. */
const RULES = [
  [/^libs\/protocol\//, ['protocol', 'core', 'desktop', 'mobile', 'sdk_go', 'desktop_fe', 'mobile_fe', ALL_PLUGINS]],
  [/^libs\/notes\//, ['core', 'desktop', 'mobile', 'sdk_go', ALL_PLUGINS]],
  [/^libs\/(core|plugin-catalog)\//, ['core', 'desktop', 'mobile']],
  [/^libs\/plugin-sdk-go\//, ['sdk_go', ALL_PLUGINS]],
  [/^libs\/plugin-sdk-ts\//, ['desktop_fe', ALL_PLUGINS]],
  [/^libs\/ui\/core\//, ['ui', 'desktop_fe', 'mobile_fe', ALL_PLUGINS]],
  [/^libs\/ui\/shell\//, ['desktop_fe']],
  [/^libs\/core-client\//, ['desktop_fe', 'mobile_fe']],
  [/^libs\/(plugin-host-ts|plugin-manager)\//, ['desktop_fe']],
  [/^apps\/desktop\/(editor|flashcards)\//, ['desktop_fe']],
  [/^apps\/desktop\/(?!editor\/|flashcards\/)/, ['desktop']],
  [/^apps\/mobile\/(bind\/|[^/]+\.go$|go\.(mod|sum)$)/, ['mobile']],
  [/^apps\/mobile\/(?!bind\/|[^/]+\.go$|go\.(mod|sum)$)/, ['mobile_fe']],
  [/^apps\/docs\//, ['docs', 'references']],
  [/^plugins\/[^/]+\/manifest\.json$/, ['core']],
  [/^plugins\//, ['plugin_versions', 'plugin_stories', 'staging']],
  [/^tests\/e2e\/desktop\//, ['desktop_fe']],
  [/^tests\/(?!e2e\/)/, ['core']],
  [/^tools\/build-plugin-index\//, ['core', 'plugin_versions']],
  [/^tools\/plugin-versions\//, ['plugin_versions']],
  [/^tools\/ci-stories\//, ['desktop_fe', 'plugin_stories']],
  [/^tools\/ci-mutation\/|^\.github\/workflows\/mutation/, ['mutation_tool']],
  [/^\.github\//, ['workflows']],
  [/^\.github\/scripts\/(stage|check-staged)/, ['staging']],
  [/(?:(?:^|\/)gradle\/wrapper\/)|(?:(?:^|\/)gradlew(?:\.bat)?$)/, ['gradle']],
  [MANUAL_SOURCES, ['docs', 'references']],
  [STORY_SOURCES, ['references']],
]

const isDocumentation = (file) => /\.mdx?$/.test(file) && !/^apps\/docs\/src\/content\//.test(file)
const isCode = (file) =>
  /^(apps\/(?!landing\/)|libs\/|plugins\/|tools\/|tests\/)/.test(file) || SHARED.test(file) || file === 'knip.json'
const isGo = (file) => /\.go$|(^|\/)go\.(mod|sum)$|^go\.work(\.sum)?$/.test(file)
const isScript = (file) => /\.(ts|tsx|js|jsx|mjs|cjs|vue)$/.test(file)

/** The plugin folder a file is in, or undefined. */
const getPluginDir = (file) => /^plugins\/([^/]+)\//.exec(file)?.[1]

/**
 * The areas of a change. `plugins` are the folders under plugins/ holding a manifest.
 *
 * @param {{ everything: boolean, files: string[], plugins: string[] }} change
 * @returns {Record<string, string>}
 */
export function getAreas({ everything, files, plugins }) {
  const areas = new Set()
  const pluginDirs = new Set()

  for (const file of everything ? [] : files) {
    if (SHARED.test(file)) {
      everything = true
      break
    }
    if (isDocumentation(file)) {
      areas.add('narrative')
      continue
    }
    if (GO_RULES.test(file)) GO_LIBS.forEach((area) => areas.add(area))
    for (const [pattern, matched] of RULES) {
      if (pattern.test(file)) matched.forEach((area) => areas.add(area))
    }
    if (isCode(file)) areas.add('code')
    if (isGo(file)) areas.add('go')
    if (isScript(file)) areas.add('js')

    const pluginDir = getPluginDir(file)
    if (pluginDir && plugins.includes(pluginDir)) {
      pluginDirs.add(pluginDir)
    } else if (pluginDir || /^plugins\/[^/]+$/.test(file)) {
      areas.add(ALL_PLUGINS)
    }
  }

  if (everything) {
    AREAS.forEach((area) => areas.add(area))
    areas.add(ALL_PLUGINS)
  }
  if (areas.has('code')) areas.add('narrative')
  if (areas.has(ALL_PLUGINS)) plugins.forEach((dir) => pluginDirs.add(dir))
  if (pluginDirs.size > 0) areas.add('plugin_stories')

  const output = Object.fromEntries(AREAS.map((area) => [area, String(areas.has(area))]))
  output.plugins = String(pluginDirs.size > 0)
  output.plugin_dirs = [...pluginDirs].sort((a, b) => a.localeCompare(b)).map((dir) => `plugins/${dir}`).join(' ')
  return output
}

/** The changed files on standard input, or undefined when there are none to read. */
const readChangedFiles = () => {
  const files = readFileSync(0, 'utf8').split('\n').filter(Boolean)
  return files.length > 0 ? files : undefined
}

const main = () => {
  const [event, input] = process.argv.slice(2)
  const files = input === '-' ? readChangedFiles() : undefined
  const plugins = readdirSync('plugins', { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && existsSync(`plugins/${entry.name}/manifest.json`))
    .map((entry) => entry.name)
  const changed = getAreas({ everything: files === undefined, files: files ?? [], plugins })
  const output = event === 'pull_request' ? changed : getAreas({ everything: true, files: [], plugins })
  // A push deploys the manual only when the push changed what the manual is written from.
  output.docs_deploy = String(event === 'push' && changed.docs === 'true')
  for (const [name, value] of Object.entries(output)) console.log(`${name}=${value}`)
  console.error(`${files?.length ?? 'all'} changed file(s)`)
}

if (import.meta.url === `file://${process.argv[1]}`) main()
