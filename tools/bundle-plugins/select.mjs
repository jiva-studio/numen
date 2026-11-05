#!/usr/bin/env node
/**
 * Selection of the plugins an application carries.
 *
 * An application is built for one target (`desktop.editor`, `desktop.flashcards`) and carries every
 * plugin the index lists for that target, whatever the plugin ships: a backend, a frontend, or only
 * contributions such as themes. The build copies the plugins this script prints and then asks it to
 * verify the built application, so the index and the folder cannot disagree.
 *
 *   node select.mjs --target <target> [--plugins <dir>]
 *       prints the folder name of each selected plugin, one per line, ordered by name
 *   node select.mjs --target <target> [--plugins <dir>] --verify <appPluginsDir>
 *       exits with an error naming each selected plugin that has no folder in <appPluginsDir>
 */
import { existsSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { buildIndex } from '../build-plugin-index/build.mjs'
import { PLUGINS_DIR, readPluginManifests } from '../shared/manifests.mjs'

/**
 * The folder names of the plugins the index lists for the target.
 *
 * @param {string} target - the application target, as the manifests name it
 * @param {string} pluginsDir - the folder holding the plugins
 * @returns {string[]} the folder names, ordered by name
 */
export const listSelectedPlugins = (target, pluginsDir = PLUGINS_DIR) => {
  const folderById = new Map(
    readPluginManifests(pluginsDir).map(({ name, manifest }) => [manifest.id, name]),
  )
  return buildIndex(pluginsDir)
    .plugins.filter((plugin) => (plugin.targets ?? []).includes(target))
    .map((plugin) => folderById.get(plugin.id))
    .sort((a, b) => a.localeCompare(b))
}

/**
 * The selected plugins that have no folder in the application.
 *
 * @param {string[]} selected - the folder names the application must hold
 * @param {string} appPluginsDir - the plugins folder of the built application
 * @returns {string[]} the missing folder names
 */
export const listMissingPlugins = (selected, appPluginsDir) =>
  selected.filter((name) => !existsSync(join(appPluginsDir, name)))

const readOption = (args, name) => (args.includes(name) ? args[args.indexOf(name) + 1] : undefined)

const main = () => {
  const args = process.argv.slice(2)
  const target = readOption(args, '--target')
  if (target === undefined) {
    console.error('usage: select.mjs --target <target> [--plugins <dir>] [--verify <appPluginsDir>]')
    process.exit(2)
  }
  const selected = listSelectedPlugins(target, readOption(args, '--plugins') ?? PLUGINS_DIR)
  const appPluginsDir = readOption(args, '--verify')
  if (appPluginsDir === undefined) {
    for (const name of selected) console.log(name)
    return
  }
  const missing = listMissingPlugins(selected, appPluginsDir)
  if (missing.length > 0) {
    for (const name of missing) console.error(`${name}: listed in the index for ${target}, not in ${appPluginsDir}`)
    process.exit(1)
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) main()
