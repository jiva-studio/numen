#!/usr/bin/env node
/**
 * Plugin index build.
 *
 * The index is one JSON file listing the manifest of every plugin of the repository, so that the
 * application knows which plugin provides a tab or a file before that plugin is on disk. Each
 * manifest is read against the schema generated from `manifest.proto` and written back in the
 * schema's JSON form: keys the schema does not name are dropped and reported as warnings.
 *
 *   node build.mjs [--plugins <dir>] [--out <file>]
 *
 * The file defaults to `plugins-index.json` in the plugins folder, which is where the application
 * looks for it next to the plugins it ships.
 */
import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { toJson } from '@bufbuild/protobuf'
import { ManifestSchema } from '../../libs/core/protocol/src/numen/v1/manifest_pb.ts'
import { PLUGINS_DIR, readPluginManifests } from '../shared/manifests.mjs'

/** The name of the index file; the application reads the same name. */
export const INDEX_FILE = 'plugins-index.json'

/** The index of every plugin under `pluginsDir`, ordered by plugin id. */
export const buildIndex = (pluginsDir = PLUGINS_DIR) => {
  const plugins = readPluginManifests(pluginsDir)
  if (plugins.length === 0) throw new Error(`no plugin with a manifest.json in ${pluginsDir}`)
  const byId = new Map()
  for (const { name, manifest, warnings } of plugins) {
    for (const warning of warnings) console.warn(`warning: plugins/${name}/manifest.json: ${warning}`)
    if (byId.has(manifest.id))
      throw new Error(`plugin id ${manifest.id} is declared twice (${name})`)
    byId.set(manifest.id, toJson(ManifestSchema, manifest))
  }
  return { plugins: [...byId.keys()].sort((a, b) => a.localeCompare(b)).map((id) => byId.get(id)) }
}

/** The text the index file holds. */
export const formatIndex = (index) => `${JSON.stringify(index, null, 2)}\n`

/** Writes the index of the plugins under `pluginsDir` to `file` and returns the index. */
export const writeIndex = (pluginsDir = PLUGINS_DIR, file = join(pluginsDir, INDEX_FILE)) => {
  const index = buildIndex(pluginsDir)
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, formatIndex(index))
  return index
}

const readOption = (args, name, fallback) =>
  args.includes(name) ? args[args.indexOf(name) + 1] : fallback

const main = () => {
  const args = process.argv.slice(2)
  const pluginsDir = readOption(args, '--plugins', PLUGINS_DIR)
  const file = readOption(args, '--out', join(pluginsDir, INDEX_FILE))
  const index = writeIndex(pluginsDir, file)
  console.log(`${index.plugins.length} plugins indexed in ${file}`)
}

if (process.argv[1] === fileURLToPath(import.meta.url)) main()
