#!/usr/bin/env node
/**
 * The mirror of the assets plugin manifests download.
 *
 *   node run.mjs list                              print the assets the manifests declare
 *   node run.mjs inventory [--write]               check inventory.json against the manifests, or rewrite it
 *   node run.mjs fetch --mirror <dir>              online: download, verify and lay out the mirror
 *   node run.mjs verify --mirror <dir>             offline: check an existing mirror against the manifests
 *   node run.mjs urls --base <url>                 print the fallback URL of each asset on a host serving the mirror
 */
import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import {
  INVENTORY_FILE,
  formatInventory,
  listAssets,
  listInventoryDrift,
  readInventory,
  readPluginDocuments,
} from './assets.mjs'
import { fillMirror, listMirrorUrls, verifyMirror } from './mirror.mjs'

const readOption = (args, name) => {
  const at = args.indexOf(name)
  return at < 0 ? '' : (args[at + 1] ?? '')
}

const runList = (assets) => {
  for (const a of assets)
    console.log(`${a.path}  ${a.sha256}  [${a.consumers.join(', ')}]\n  ${a.urls.join('\n  ')}`)
  return 0
}

const runInventory = (assets, args) => {
  if (args.includes('--write')) {
    writeFileSync(INVENTORY_FILE, formatInventory(assets))
    console.log(`wrote ${assets.length} assets to inventory.json`)
    return 0
  }
  const problems = listInventoryDrift(assets, readInventory())
  for (const problem of problems) console.error(problem)
  if (problems.length > 0) console.error('run: node tools/plugin-mirror/run.mjs inventory --write')
  return problems.length > 0 ? 1 : 0
}

const runVerify = (assets, root) => {
  const { problems, extras } = verifyMirror(root, assets)
  for (const problem of problems) console.error(problem)
  for (const extra of extras) console.warn(`${extra} is in the mirror and no manifest names it`)
  console.log(`${assets.length - problems.length} of ${assets.length} assets verified`)
  return problems.length > 0 ? 1 : 0
}

const runFetch = async (assets, root) => {
  const result = await fillMirror(root, assets, (url) => fetch(url))
  for (const line of result.failed) console.error(`unavailable: ${line}`)
  console.log(
    `${result.mirrored.length} mirrored, ${result.kept.length} already verified, ${result.failed.length} failed`,
  )
  return result.failed.length > 0 ? 1 : 0
}

const runUrls = (assets, base) => {
  for (const one of listMirrorUrls(base, assets)) console.log(`${one.path}  ${one.url}`)
  return 0
}

export const run = async (args) => {
  const assets = listAssets(readPluginDocuments())
  const root = readOption(args, '--mirror')
  const base = readOption(args, '--base')
  const commands = {
    list: () => runList(assets),
    inventory: () => runInventory(assets, args),
    verify: () => (root ? runVerify(assets, root) : usage()),
    fetch: () => (root ? runFetch(assets, root) : usage()),
    urls: () => (base ? runUrls(assets, base) : usage()),
  }
  return (commands[args[0]] ?? usage)()
}

const usage = () => {
  console.error(
    'usage: run.mjs list | inventory [--write] | fetch --mirror <dir> | verify --mirror <dir> | urls --base <url>',
  )
  return 2
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  process.exitCode = await run(process.argv.slice(2))
}
