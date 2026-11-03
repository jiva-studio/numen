/**
 * Prints, as `name=value` lines for $GITHUB_OUTPUT, which areas of the repository a change touches
 * and so which CI jobs run.
 *
 * Usage: node changes.mjs <event> <base> <head>
 *
 * A pull request runs the areas its files touch and every area that depends on them. A push, a
 * manual run and a change to code every area reads (SHARED) run every area. `plugin_dirs` lists
 * the plugin folders to lint, build and test; `e2e` is `all`, `none` or the folders under
 * tests/e2e/desktop/tests/plugins/ to run.
 */
import { execFileSync } from 'node:child_process'

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
const ALL_E2E = 'e2e'

/** Read by every area: the workspace, the build, the shared tools and CI itself. */
const SHARED =
  /^(go\.work(\.sum)?|package(-lock)?\.json|Makefile|\.node-version|tsconfig\.base\.json|prettier\.config\.mjs|\.prettierignore|stylelint\.config\.mjs)$|^tools\/shared\/|^tools\/lint-layers\/base-eslint\.mjs$|^\.github\/actions\/|^\.github\/scripts\/changes|^\.github\/workflows\/(ci|changes)\.yml$/

/** The Go lint rules every Go module is held to. */
const GO_RULES = /^\.golangci(-structure)?\.yml$|(^|\/)gorules\//

const GO_LIBS = ['core', 'desktop', 'mobile', 'sdk_go', ALL_PLUGINS, ALL_E2E]

/** What the manual writes itself from, besides apps/docs. */
const MANUAL_SOURCES =
  /^apps\/desktop\/editor\/src\/(features\/command-palette\/lib\/|features\/command-palette\/model\/palette\/shortcuts\.ts$|shared\/words\.ts$|app\/commands\/fileMenuCommands\.ts$|app\/window\/useSettings\.ts$)|^apps\/desktop\/cmd\/numen\/main\.go$|^libs\/core\/(adapter\/cli\/cli\.go$|container\/(settings|indexing)\.go$|adapter\/settings\/|usecase\/completion\/)|^plugins\/[^/]+\/(manifest\.json$|frontend\/index\.ts$|frontend\/.*words\.ts$)|^tools\/ci-commands\//

/** Story names the manual and the landing page hold, and the trees they are held against. */
const STORY_SOURCES =
  /^libs\/ui\/(core|shell)\/src\/|^apps\/desktop\/(editor|flashcards)\/src\/|^plugins\/[^/]+\/frontend\/|^plugins\/themes-presets\/themes\/|^tools\/ci-(stories|commands)\/|^apps\/docs\/(tools|src\/content)\/|^apps\/landing\/tools\//

/** Each rule adds its areas to a file it matches; a file can match several. */
const RULES = [
  [/^libs\/protocol\//, ['protocol', 'core', 'desktop', 'mobile', 'sdk_go', 'desktop_fe', 'mobile_fe', ALL_PLUGINS, ALL_E2E]],
  [/^libs\/notes\//, ['core', 'desktop', 'mobile', 'sdk_go', ALL_PLUGINS, ALL_E2E]],
  [/^libs\/(core|plugin-catalog)\//, ['core', 'desktop', 'mobile', ALL_E2E]],
  [/^libs\/plugin-sdk-go\//, ['sdk_go', ALL_PLUGINS, ALL_E2E]],
  [/^libs\/plugin-sdk-ts\//, ['desktop_fe', ALL_PLUGINS, ALL_E2E]],
  [/^libs\/ui\/core\//, ['ui', 'desktop_fe', 'mobile_fe', ALL_PLUGINS, ALL_E2E]],
  [/^libs\/ui\/shell\//, ['desktop_fe', ALL_E2E]],
  [/^libs\/core-client\//, ['desktop_fe', 'mobile_fe', ALL_E2E]],
  [/^libs\/(plugin-host-ts|plugin-manager)\//, ['desktop_fe', ALL_E2E]],
  [/^apps\/desktop\/(editor|flashcards)\//, ['desktop_fe', ALL_E2E]],
  [/^apps\/desktop\/(?!editor\/|flashcards\/)/, ['desktop', ALL_E2E]],
  [/^apps\/mobile\/(bind\/|[^/]+\.go$|go\.(mod|sum)$)/, ['mobile']],
  [/^apps\/mobile\/(?!bind\/|[^/]+\.go$|go\.(mod|sum)$)/, ['mobile_fe']],
  [/^apps\/docs\//, ['docs']],
  [/^plugins\/[^/]+\/manifest\.json$/, ['core']],
  [/^plugins\//, ['plugin_versions', 'plugin_stories', 'staging']],
  [/^tests\/e2e\/desktop\//, ['desktop_fe']],
  [/^tests\/(?!e2e\/)/, ['core', ALL_E2E]],
  [/^tools\/build-plugin-index\//, ['core', 'plugin_versions', ALL_E2E]],
  [/^tools\/plugin-versions\//, ['plugin_versions']],
  [/^tools\/ci-stories\//, ['desktop_fe', 'plugin_stories']],
  [/^tools\/ci-mutation\/|^\.github\/workflows\/mutation/, ['mutation_tool']],
  [/^\.github\//, ['workflows']],
  [/^\.github\/scripts\/(stage|check-staged)/, ['staging']],
  [/(^|\/)gradle\/wrapper\/|(^|\/)gradlew(\.bat)?$/, ['gradle']],
  [MANUAL_SOURCES, ['docs']],
  [STORY_SOURCES, ['references']],
]

const isDocumentation = (file) => /\.mdx?$/.test(file) && !/^apps\/docs\/src\/content\//.test(file)
const isCode = (file) =>
  /^(apps\/(?!landing\/)|libs\/|plugins\/|tools\/|tests\/)/.test(file) || SHARED.test(file) || file === 'knip.json'
const isGo = (file) => /\.go$|(^|\/)go\.(mod|sum)$|^go\.work(\.sum)?$/.test(file)
const isScript = (file) => /\.(ts|tsx|js|jsx|mjs|cjs|vue)$/.test(file)

/** The plugin folder a file is in, or undefined. */
const getPluginDir = (file) => /^plugins\/([^/]+)\//.exec(file)?.[1]

/** The E2E folder of a plugin: its folder name with dots for dashes, when the suite has one. */
const getE2eFolder = (pluginDir, e2eFolders) => {
  const folder = pluginDir.replaceAll('-', '.')
  return e2eFolders.includes(folder) ? folder : undefined
}

/**
 * The areas of a change. `plugins` are the folders under plugins/ holding a manifest, and
 * `e2eFolders` the folders under tests/e2e/desktop/tests/plugins/.
 *
 * @param {{ everything: boolean, files: string[], plugins: string[], e2eFolders: string[] }} change
 * @returns {Record<string, string>}
 */
export function getAreas({ everything, files, plugins, e2eFolders }) {
  const areas = new Set()
  const pluginDirs = new Set()
  const e2e = new Set()

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
      const folder = getE2eFolder(pluginDir, e2eFolders)
      if (folder) e2e.add(folder)
    } else if (pluginDir || /^plugins\/[^/]+$/.test(file)) {
      areas.add(ALL_PLUGINS)
    }
    const e2eFolder = /^tests\/e2e\/desktop\/tests\/plugins\/([^/]+)\//.exec(file)?.[1]
    if (e2eFolder) e2e.add(e2eFolder)
    else if (/^tests\/e2e\/desktop\//.test(file)) areas.add(ALL_E2E)
  }

  if (everything) {
    AREAS.forEach((area) => areas.add(area))
    areas.add(ALL_PLUGINS)
    areas.add(ALL_E2E)
  }
  if (areas.has('code')) areas.add('narrative')
  if (areas.has(ALL_PLUGINS)) plugins.forEach((dir) => pluginDirs.add(dir))
  if (pluginDirs.size > 0) areas.add('plugin_stories')

  const output = Object.fromEntries(AREAS.map((area) => [area, String(areas.has(area))]))
  output.plugins = String(pluginDirs.size > 0)
  output.plugin_dirs = [...pluginDirs].sort().map((dir) => `plugins/${dir}`).join(' ')
  output.e2e = areas.has(ALL_E2E) ? 'all' : e2e.size > 0 ? [...e2e].sort().join(' ') : 'none'
  return output
}

/** The paths git lists under `path` at HEAD, one level deep. */
const listTree = (path) =>
  execFileSync('git', ['ls-tree', '--name-only', 'HEAD', `${path}/`], { encoding: 'utf8' })
    .split('\n')
    .filter(Boolean)
    .map((entry) => entry.slice(path.length + 1))

const main = () => {
  const [event, base, head] = process.argv.slice(2)
  const everything = event !== 'pull_request'
  const files = everything
    ? []
    : execFileSync('git', ['diff', '--name-only', '--no-renames', base, head], { encoding: 'utf8' })
        .split('\n')
        .filter(Boolean)
  const manifests = execFileSync('git', ['ls-tree', '-r', '--name-only', 'HEAD', 'plugins/'], { encoding: 'utf8' })
    .split('\n')
    .map((path) => /^plugins\/([^/]+)\/manifest\.json$/.exec(path)?.[1])
    .filter(Boolean)
  const output = getAreas({
    everything,
    files,
    plugins: manifests,
    e2eFolders: listTree('tests/e2e/desktop/tests/plugins'),
  })
  for (const [name, value] of Object.entries(output)) console.log(`${name}=${value}`)
  console.error(`${files.length} changed file(s)`)
}

if (import.meta.url === `file://${process.argv[1]}`) main()
