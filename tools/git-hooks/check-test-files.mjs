#!/usr/bin/env node
/**
 * A pull request of type `test` changes test material only.
 *
 *   PR_TITLE='test(core): ...' node check-test-files.mjs [--base <ref>]
 *
 * The base defaults to HEAD^1, the first parent of the merge commit CI checks out. A pull request
 * of another type passes unread. A changed file that is not test material passes when its diff
 * only adds, removes or edits mutation directives, or, for a `manifest.json` or `package.json`,
 * only changes `"version"`.
 */
import path from 'node:path'
import process from 'node:process'
import { isTestSupport } from '../ci-mutation/targets.mjs'
import { execTool } from '../shared/tools.mjs'

const TEST_FILE = /(_test\.go|\.(test|spec)\.[cm]?[jt]sx?|\.stories\.ts|\.feature)$/
const TEST_FOLDER = /(^|\/)(testdata|test)\//
const TEST_ROOT = /^(tests|features)\//
const STRYKER_CONFIG = /(^|\/)stryker\.(config|conf)\.([cm]?[jt]s|json)$/
const VERSIONED_FILE = /(^|\/)(manifest|package)\.json$/
const DIRECTIVE = String.raw`(?:\/\/|\/\*)\s*(?:Stryker\s+(?:disable|restore)\b|gomutants:)`
const DIRECTIVE_LINE = new RegExp(String.raw`^\s*${DIRECTIVE}.*$`)
const TRAILING_DIRECTIVE = new RegExp(String.raw`\s*${DIRECTIVE}.*$`)

/** The Conventional Commits type of a title, or null. */
export const getTitleType = (title) => /^([a-z]+)(?:\([^)]*\))?!?:/.exec(title.trim())?.[1] ?? null

export const isTestMaterial = (file) =>
  TEST_FILE.test(file) ||
  TEST_FOLDER.test(file) ||
  TEST_ROOT.test(file) ||
  STRYKER_CONFIG.test(file) ||
  isTestSupport(file)

/** The text with every Stryker and gomutants directive comment taken out. */
export const removeDirectives = (text) =>
  text
    .split('\n')
    .filter((line) => !DIRECTIVE_LINE.test(line))
    .map((line) => line.replace(TRAILING_DIRECTIVE, ''))
    .join('\n')

const parseJson = (text) => {
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

export const isVersionOnlyChange = (before, after) => {
  const [a, b] = [parseJson(before), parseJson(after)]
  if (a === null || b === null || typeof a !== 'object' || typeof b !== 'object') return false
  return JSON.stringify({ ...a, version: null }) === JSON.stringify({ ...b, version: null })
}

/**
 * The changed files that are not test material. `changes` are `{ status, file }` pairs of a diff
 * without renames; `readBase` and `readHead` return a file's text on either side.
 */
export const findProductionChanges = ({ changes, readBase, readHead }) =>
  changes
    .filter(({ file }) => !isTestMaterial(file))
    .filter(({ status, file }) => {
      if (status !== 'M') return true
      const [before, after] = [readBase(file), readHead(file)]
      if (VERSIONED_FILE.test(file) && isVersionOnlyChange(before, after)) return false
      return removeDirectives(before) !== removeDirectives(after)
    })
    .map(({ file }) => file)

const readChanges = (base) =>
  execTool('git', ['diff', '--no-renames', '--name-status', base, 'HEAD'], { encoding: 'utf8' })
    .split('\n')
    .filter(Boolean)
    .map((line) => {
      const [status, file] = line.split('\t')
      return { status: status[0], file }
    })

const readAt = (ref) => (file) =>
  execTool('git', ['show', `${ref}:${file}`], { encoding: 'utf8', maxBuffer: 1 << 28 })

function main() {
  const args = process.argv.slice(2)
  const base = args.includes('--base') ? args[args.indexOf('--base') + 1] : 'HEAD^1'
  const title = process.env.PR_TITLE ?? ''
  if (getTitleType(title) !== 'test') return
  const files = findProductionChanges({
    changes: readChanges(base),
    readBase: readAt(base),
    readHead: readAt('HEAD'),
  })
  if (files.length === 0) return
  console.error(
    [
      `\n❌ Error: a test pull request changes files that are not tests:`,
      ...files.map((file) => `  ${file}`),
      '',
      'Kill a mutant with a new or stronger test. An equivalent mutant gets a directive with its reason;',
      'a change to production code goes into a separate pull request of another type.',
      '',
    ].join('\n'),
  )
  process.exit(1)
}

const isDirectRun =
  process.argv[1] && new URL(import.meta.url).pathname === path.resolve(process.argv[1])
if (isDirectRun) {
  main()
}
