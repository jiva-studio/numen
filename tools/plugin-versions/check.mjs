#!/usr/bin/env node
/**
 * Plugin version check.
 *
 * A plugin version is `<year>.<month>.<build>` (`2026.10.3`): the year and the
 * month of the change, and the number of the change within that month,
 * starting at 1. The result is valid semver, which the host requires.
 *
 *   node check.mjs [--base <ref>]            fail when a changed plugin keeps its version
 *   node check.mjs [--base <ref>] --write    write the next version into each changed plugin
 */
import { execFileSync, spawnSync } from 'node:child_process'
import { readFileSync, existsSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { MANIFEST_FILE, listPluginNames, parseManifest } from '../shared/manifests.mjs'
import { readPackageVersion } from './package.mjs'

const VERSION_PATTERN = /^(\d{4})\.([1-9]|1[0-2])\.([1-9]\d*)$/
const VERSION_FIELD = /("version"\s*:\s*")[^"]*(")/
const IGNORED_FILE = /(\.md|\.test\.[cm]?[jt]sx?|_test\.go)$/

export const parseVersion = (text) => {
  const match = VERSION_PATTERN.exec(text ?? '')
  return match ? { year: Number(match[1]), month: Number(match[2]), build: Number(match[3]) } : null
}

export const compareVersions = (a, b) => a.year - b.year || a.month - b.month || a.build - b.build

export const formatVersion = ({ year, month, build }) => `${year}.${month}.${build}`

/** The version a change made on `today` takes after `base`. */
export const getNextVersion = (base, today) => {
  const parsed = parseVersion(base)
  const year = today.getUTCFullYear()
  const month = today.getUTCMonth() + 1
  if (parsed && parsed.year === year && parsed.month === month) {
    return formatVersion({ year, month, build: parsed.build + 1 })
  }
  return formatVersion({ year, month, build: 1 })
}

/** The files of a plugin whose change alters the plugin. */
export const getCodeFiles = (files) => files.filter((file) => !IGNORED_FILE.test(file))

/** The reason `current` is wrong for a plugin, or null. */
export const checkVersion = ({ current, base, isChanged, today }) => {
  const parsed = parseVersion(current)
  if (!parsed) return `version "${current}" is not <year>.<month>.<build>`
  const limit = parseVersion(getNextVersion('', today))
  if (compareVersions({ ...parsed, build: 1 }, limit) > 0) {
    return `version ${current} is later than the current month`
  }
  const parsedBase = parseVersion(base)
  if (isChanged && parsedBase && compareVersions(parsed, parsedBase) <= 0) {
    return `plugin changed, version ${current} was not raised (next: ${getNextVersion(base, today)})`
  }
  return null
}

const git = (...args) => execFileSync('git', args, { encoding: 'utf8' }).trim()

/** The version a plugin had on `base`, and null where the plugin did not exist there. */
const readBaseVersion = (base, dir) => {
  const source = `${base}:${dir}/${MANIFEST_FILE}`
  const shown = spawnSync('git', ['show', source], { encoding: 'utf8' })
  return shown.status === 0 ? parseManifest(shown.stdout, source).manifest.version : null
}

const main = () => {
  const args = process.argv.slice(2)
  const baseRef = args.includes('--base') ? args[args.indexOf('--base') + 1] : 'origin/main'
  const isWrite = args.includes('--write')
  const root = fileURLToPath(new URL('../../', import.meta.url))
  process.chdir(root)

  const mergeBase = git('merge-base', 'HEAD', baseRef)
  const today = new Date()
  const failures = []

  for (const name of listPluginNames('plugins')) {
    const dir = `plugins/${name}`
    const manifestPath = join(dir, MANIFEST_FILE)

    const text = readFileSync(manifestPath, 'utf8')
    const current = parseManifest(text, manifestPath).manifest.version
    const base = readBaseVersion(mergeBase, dir)
    const changed = git('diff', '--name-only', mergeBase, '--', dir).split('\n').filter(Boolean)
    const isChanged = getCodeFiles(changed).length > 0

    const packagePath = join(dir, 'package.json')
    const packageText = existsSync(packagePath) ? readFileSync(packagePath, 'utf8') : null
    const packageVersion = packageText === null ? current : readPackageVersion(packageText)

    const error = checkVersion({ current, base, isChanged, today })
    if (isWrite && isChanged && error) {
      const next = getNextVersion(base ?? '', today)
      writeFileSync(manifestPath, text.replace(VERSION_FIELD, `$1${next}$2`))
      if (packageText !== null)
        writeFileSync(packagePath, packageText.replace(VERSION_FIELD, `$1${next}$2`))
      console.log(`${name}: ${current} -> ${next}`)
    } else if (error) {
      failures.push(`${name}: ${error}`)
    } else if (packageVersion !== current) {
      failures.push(
        `${name}: package.json version ${packageVersion} differs from manifest.json ${current}`,
      )
    }
  }

  if (failures.length > 0) {
    console.error(failures.join('\n'))
    console.error('\nRun `make plugin_versions_bump` to write the next versions.')
    process.exit(1)
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) main()
