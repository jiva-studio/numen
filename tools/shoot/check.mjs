/**
 * Holds the declared screenshots against the files: every story tagged `screenshot` has its two
 * files, and every file under `plugins/<folder>/screenshots/` is written by a declared shot.
 *
 * Usage: node tools/shoot/check.mjs
 */
import { existsSync } from 'node:fs'
import { readdir } from 'node:fs/promises'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { root } from '../shared/modules.mjs'
import { getFiles, getPluginFolder, readShots } from './shots.mjs'

/**
 * The faults of `shots` against the files: `present` holds every file that exists, relative to the
 * repository, and `pluginFiles` the files found under the plugins' screenshot folders.
 */
export const getFileFaults = (shots, present, pluginFiles) => {
  const faults = []
  if (shots.length === 0) faults.push('no story declares a screenshot')
  const written = new Set()
  for (const shot of shots) {
    for (const file of getFiles(shot)) {
      written.add(file)
      if (!present.has(file)) {
        faults.push(`${file} is missing: run node tools/shoot/shoot.mjs --story ${shot.story}`)
      }
    }
  }
  for (const file of pluginFiles) {
    if (!written.has(file)) faults.push(`${file} is written by no story tagged screenshot`)
  }
  return faults
}

const listPluginFiles = async () => {
  const files = []
  for (const plugin of await readdir(join(root, 'plugins'))) {
    const folder = getPluginFolder(plugin)
    if (!existsSync(join(root, folder))) continue
    for (const name of await readdir(join(root, folder))) files.push(`${folder}/${name}`)
  }
  return files
}

const main = async () => {
  const { shots, faults } = await readShots()
  const pluginFiles = await listPluginFiles()
  const present = new Set(
    [...shots.flatMap(getFiles), ...pluginFiles].filter((file) => existsSync(join(root, file))),
  )
  faults.push(...getFileFaults(shots, present, pluginFiles))
  if (faults.length > 0) {
    process.stderr.write(`screenshots: ${faults.join('\n')}\n`)
    process.exit(1)
  }
  process.stdout.write(`screenshots: ${shots.length} shots, each with its files\n`)
}

if (process.argv[1] === fileURLToPath(import.meta.url)) await main()
