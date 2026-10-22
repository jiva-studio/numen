import { readdirSync, statSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { pluginCatalog } from './pluginCatalog'

const pluginsRoot = resolve(process.cwd(), '../../../plugins')

function listFrontendPlugins(): string[] {
  return readdirSync(pluginsRoot).filter((name) => {
    const frontend = join(pluginsRoot, name, 'frontend')
    try {
      return statSync(frontend).isDirectory()
    } catch {
      return false
    }
  })
}

describe('pluginCatalog', () => {
  it('has a loader for every plugin directory that has a frontend directory', () => {
    const directories = listFrontendPlugins()
    expect(directories).toContain('engines.embeddings')
    expect(Object.keys(pluginCatalog.loaders)).toHaveLength(directories.length)
  })

  // One test per plugin: each frontend's import graph is loaded under its own
  // timeout, so the cost of the whole catalog is not charged to a single test.
  it.each(Object.keys(pluginCatalog.loaders))('loads the frontend of %s', async (id) => {
    expect(pluginCatalog.manifests[id]?.id).toBe(id)
    const loaded = (await pluginCatalog.loaders[id]?.()) as {
      activate?: unknown
      default?: { activate?: unknown }
    }
    expect(typeof (loaded.activate ?? loaded.default?.activate)).toBe('function')
  })

  it('takes the permissions of a plugin from its manifest.json', () => {
    expect(pluginCatalog.manifests['md.numen.tools.media.importer']?.permissions).toStrictEqual([
      'network:outbound',
      'vault:write',
    ])
  })
})
