/**
 * What the syntax trees of Go files show, read by `tools/plugin-registry/gosource`, which is built
 * on its own go.mod: per file, its test and fuzz functions, its network client calls resolved by
 * import path, and its URL and `host:port` string literals.
 */
import { fileURLToPath } from 'node:url'
import { runTool } from '../shared/tools.mjs'

const HELPER = fileURLToPath(new URL('./gosource/', import.meta.url))

const MAX_BUFFER = 64 * 1024 * 1024

/**
 * @typedef {{ tests: number, calls: { name: string, target?: string }[], urls: string[], addresses: string[] }} GoFileFacts
 */

/**
 * Reads the Go files `files` and throws where Go is not installed or the reader fails.
 *
 * @param {string[]} files - absolute paths
 * @returns {Map<string, GoFileFacts>} by path
 */
export const readGoFiles = (files) => {
  if (files.length === 0) return new Map()
  const read = runTool('go', ['run', '.'], {
    cwd: HELPER,
    env: { ...process.env, GOWORK: 'off' },
    input: files.join('\n'),
    encoding: 'utf8',
    maxBuffer: MAX_BUFFER,
  })
  if (read.error) throw read.error
  if (read.status !== 0) throw new Error(`tools/plugin-registry/gosource failed: ${read.stderr}`)
  return new Map(Object.entries(JSON.parse(read.stdout)))
}
