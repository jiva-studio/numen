/**
 * The one place the tools start an external program.
 *
 * A program is looked up in the directories of PATH that are trusted, never in the others: a
 * relative entry or a world-writable folder cannot put its own `git` in front of the real one. The
 * trusted directories are the system ones, the Nix store and profiles, the Node and runner tool
 * caches, the node_modules/.bin folders of this repository, the home folders of the Go and Nix
 * tools, and the entries of NUMEN_TOOL_PATH (a list separated like PATH).
 */
import { spawn, spawnSync, execFileSync } from 'node:child_process'
import { accessSync, constants, statSync } from 'node:fs'
import { homedir } from 'node:os'
import { delimiter, dirname, isAbsolute, join, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const REPO_ROOT = fileURLToPath(new URL('../../', import.meta.url))

const TRUSTED_DIRECTORIES = [
  '/usr/bin',
  '/bin',
  '/usr/local/bin',
  '/opt/homebrew/bin',
  '/run/current-system/sw/bin',
]

/** Folders whose whole subtree is trusted, because a package manager or a runner image owns them. */
const trustedPrefixes = (env) =>
  [
    '/nix/store',
    '/nix/var/nix/profiles',
    '/opt/hostedtoolcache',
    '/usr/local',
    '/usr/bin',
    '/opt/homebrew',
    env.RUNNER_TOOL_CACHE,
    join(homedir(), '.nix-profile'),
    join(homedir(), 'go', 'bin'),
    dirname(process.execPath),
    REPO_ROOT,
  ].filter(Boolean)

const isInside = (dir, prefix) => dir === prefix || dir.startsWith(prefix.endsWith(sep) ? prefix : prefix + sep)

/** Whether a PATH entry is an absolute directory this module trusts. */
export const isTrustedDirectory = (dir, env = process.env) => {
  if (!isAbsolute(dir)) return false
  const extra = (env.NUMEN_TOOL_PATH ?? '').split(delimiter).filter(Boolean)
  return [...TRUSTED_DIRECTORIES, ...extra].includes(dir) || trustedPrefixes(env).some((prefix) => isInside(dir, prefix))
}

const isExecutable = (file) => {
  try {
    if (!statSync(file).isFile()) return false
    accessSync(file, constants.X_OK)
    return true
  } catch {
    return false
  }
}

/** The absolute path of the program `name` in the first trusted directory of PATH that holds it, or null. */
export const findTool = (name, env = process.env) => {
  const dirs = [...(env.PATH ?? '').split(delimiter), ...(env.NUMEN_TOOL_PATH ?? '').split(delimiter)]
  for (const dir of dirs.filter(Boolean)) {
    if (!isTrustedDirectory(dir, env)) continue
    const candidate = join(dir, name)
    if (isExecutable(candidate)) return candidate
  }
  return null
}

const getMissingMessage = (name) =>
  `${name} is not in a trusted directory of PATH (add its directory to NUMEN_TOOL_PATH to allow it)`

/** Runs the program like spawnSync. A program that is not found is reported in `error`, as spawnSync does. */
export const runTool = (name, args, options) => {
  const file = findTool(name)
  if (!file) return { error: new Error(getMissingMessage(name)), status: null, stdout: '', stderr: '' }
  return spawnSync(file, args, options)
}

/** Runs the program like execFileSync and throws where it is not found. */
export const execTool = (name, args, options) => {
  const file = findTool(name)
  if (!file) throw new Error(getMissingMessage(name))
  return execFileSync(file, args, options)
}

/** Starts the program like spawn. A program that is not found throws. */
export const spawnTool = (name, args, options) => {
  const file = findTool(name)
  if (!file) throw new Error(getMissingMessage(name))
  return spawn(file, args, options)
}
