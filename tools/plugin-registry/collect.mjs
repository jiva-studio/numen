#!/usr/bin/env node
/**
 * Plugin registry collector.
 *
 * Reads every plugin of `plugins/` and every external source of `plugins/registry.json`, and
 * writes the registry the site's plugin catalog is built from:
 *
 *   <out>/latest.json                  the registry
 *   <out>/<YYYY-MM-DD>.json            the same registry, kept by date
 *   <out>/<id>/screenshots/*.webp      screenshots of the external plugins fetched this run
 *
 * The folder is published at `plugins/registry/` of the downloads zone. With `--previous`, external
 * releases that did not change are copied from it, download sizes it knows are not asked again,
 * and the content hash is compared: `changed=false` (in GITHUB_OUTPUT when set) means the
 * registry is the one already published.
 *
 *   node collect.mjs --out <dir> [--previous <latest.json>] [--quality <dir or https URL>]
 *                    [--sources <file>] [--offline]
 *
 * `--quality` holds `mutation.json` and `coverage.json` (`tools/shared/plugin-quality.mjs`).
 * `--offline` asks nothing over the network: download sizes are unknown, and external plugins keep
 * their previous entries.
 */
import { appendFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { listPluginNames } from '../shared/manifests.mjs'
import { fetchSize, listKnownSizes } from './downloads.mjs'
import { collectExternal, createGitHub } from './external.mjs'
import { collectInternal, readRepositoryLicense } from './internal.mjs'
import { readQualityFiles } from './quality.mjs'
import {
  assertFeaturedKnown,
  assertNoCollision,
  createRegistry,
  formatRegistry,
  readFeatured,
  readSources,
} from './registry.mjs'

const ROOT = fileURLToPath(new URL('../../', import.meta.url))

/** The name of the registry file the site reads. */
export const LATEST_FILE = 'latest.json'

const readPrevious = (file) =>
  file && existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')) : null

const collectExternals = async (sources, context, isOffline) => {
  const entries = []
  for (const source of sources) {
    const previous = context.previousById.get(source.id)
    if (isOffline) {
      if (previous) entries.push(previous)
      continue
    }
    try {
      entries.push(await collectExternal(source, context))
    } catch (error) {
      console.warn(`::warning::${source.id}: ${error.message}`)
      if (previous) entries.push(previous)
    }
  }
  return entries
}

/**
 * Collects the registry.
 *
 * @param {object} options
 * @param {string} options.root - the repository root
 * @param {string} options.outDir - where external screenshots are written
 * @param {object | null} options.previous - the previous registry
 * @param {string | null} options.quality - the quality folder or URL
 * @param {string} options.sources - the external sources file
 * @param {boolean} options.isOffline
 * @param {typeof fetch} [options.fetcher]
 * @param {object} [options.verifiers] - the attestation and vulnerability checks of external plugins
 * @param {string} [options.generatedAt]
 * @returns {Promise<object>} the registry
 */
export const collectRegistry = async ({
  root,
  outDir,
  previous,
  quality,
  sources,
  isOffline,
  fetcher = fetch,
  verifiers,
  generatedAt = new Date().toISOString(),
}) => {
  const previousById = new Map((previous?.plugins ?? []).map((one) => [one.id, one]))
  const context = {
    root,
    outDir,
    previousById,
    license: readRepositoryLicense(root),
    knownSizes: listKnownSizes(previous),
    getSize: isOffline ? async () => null : (url) => fetchSize(url, fetcher),
    quality: await readQualityFiles(quality, fetcher),
    github: createGitHub({ token: process.env.GITHUB_TOKEN || process.env.GH_TOKEN, fetcher }),
    ...(verifiers ? { verifiers } : {}),
  }
  const folders = listPluginNames(join(root, 'plugins'))
  const internals = []
  for (const folder of folders) internals.push(await collectInternal(folder, context))
  const externalSources = readSources(sources)
  assertNoCollision(
    externalSources,
    internals.map((one, index) => ({ id: one.id, folder: folders[index] })),
  )
  const externals = await collectExternals(externalSources, context, isOffline)
  const plugins = [...internals, ...externals]
  const featured = readFeatured(sources)
  assertFeaturedKnown(featured, plugins)
  return createRegistry(plugins, generatedAt, featured)
}

const readOption = (args, name, fallback) =>
  args.includes(name) ? args[args.indexOf(name) + 1] : fallback

const main = async () => {
  const args = process.argv.slice(2)
  const outDir = readOption(args, '--out', null)
  if (!outDir) throw new Error('usage: collect.mjs --out <dir> [--previous <file>] [--quality <dir|url>]')
  const previous = readPrevious(readOption(args, '--previous', null))
  const registry = await collectRegistry({
    root: ROOT,
    outDir,
    previous,
    quality: readOption(args, '--quality', null),
    sources: readOption(args, '--sources', join(ROOT, 'plugins', 'registry.json')),
    isOffline: args.includes('--offline'),
  })
  const isChanged = registry.contentHash !== previous?.contentHash
  mkdirSync(outDir, { recursive: true })
  writeFileSync(join(outDir, LATEST_FILE), formatRegistry(registry))
  writeFileSync(join(outDir, `${registry.generatedAt.slice(0, 10)}.json`), formatRegistry(registry))
  if (process.env.GITHUB_OUTPUT) {
    appendFileSync(process.env.GITHUB_OUTPUT, `changed=${isChanged}\ndate=${registry.generatedAt.slice(0, 10)}\n`)
  }
  console.log(
    `${registry.plugins.length} plugins, content ${registry.contentHash.slice(0, 12)}, ${isChanged ? 'changed' : 'unchanged'}`,
  )
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main().catch((error) => {
    console.error(error.message)
    process.exit(1)
  })
}
