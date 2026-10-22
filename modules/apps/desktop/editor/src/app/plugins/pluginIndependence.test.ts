/**
 * Frontend architectural boundary tests that keep the host and plugins independent.
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { useWindowTabs } from '@/entities/tab'
import { getTabs, mountWindow } from '@/testing/window'
import { pluginCatalog } from './pluginCatalog'

const editorSrcDir = resolve(import.meta.dirname, '../..')
const pluginsDir = resolve(import.meta.dirname, '../../../../../../plugins')

/** Every plugin id from the manifests in modules/plugins. */
function listPluginIds(): string[] {
  const ids: string[] = []
  for (const entry of readdirSync(pluginsDir)) {
    const manifestPath = join(pluginsDir, entry, 'manifest.json')
    try {
      if (statSync(manifestPath).isFile()) {
        const manifest = JSON.parse(readFileSync(manifestPath, 'utf8')) as { id?: string }
        if (manifest.id) ids.push(manifest.id)
      }
    } catch {
      // Skip non-plugin directories
    }
  }
  return ids
}

/** Lists all files under dir recursively, skipping tests, stories, and testing mocks. */
function listHostSources(dir: string): string[] {
  const sources: string[] = []
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    const stat = statSync(full)
    if (stat.isDirectory()) {
      if (entry === 'testing' || entry === 'node_modules') continue
      sources.push(...listHostSources(full))
    } else if (stat.isFile() && (entry.endsWith('.ts') || entry.endsWith('.vue'))) {
      if (entry.includes('.test.') || entry.includes('.spec.') || entry.includes('.stories.')) {
        continue
      }
      sources.push(full)
    }
  }
  return sources
}

const pluginIdRegex = /md\.numen\.[a-zA-Z0-9_-]+/
const readerKindRegex = /['"]reader:[a-zA-Z0-9_-]+['"]/

function findForbiddenLiterals(content: string, pluginIds: readonly string[]): string[] {
  const found: string[] = []
  const idMatch = content.match(pluginIdRegex)
  if (idMatch) {
    found.push(`spells out plugin id ${idMatch[0]}`)
  }
  for (const id of pluginIds) {
    if (content.includes(id) && !found.some((msg) => msg.includes(id))) {
      found.push(`spells out plugin id ${id}`)
    }
  }
  const readerMatch = content.match(readerKindRegex)
  if (readerMatch) {
    found.push(`spells out reader tab kind ${readerMatch[0]}`)
  }
  return found
}

describe('host and plugin independence', () => {
  it('host sources outside the composition file name no plugin id or plugin tab kind', () => {
    const pluginIds = listPluginIds()
    expect(pluginIds.length).toBeGreaterThan(0)

    const hostSources = listHostSources(editorSrcDir)
    expect(hostSources.length).toBeGreaterThan(10)

    const compositionFile = resolve(editorSrcDir, 'app/plugins/pluginCatalog.ts')
    const violations: { file: string; issues: string[] }[] = []

    for (const source of hostSources) {
      if (source === compositionFile) continue
      const content = readFileSync(source, 'utf8')
      const issues = findForbiddenLiterals(content, pluginIds)
      if (issues.length > 0) {
        violations.push({ file: source.replace(editorSrcDir, 'src'), issues })
      }
    }

    expect(violations).toStrictEqual([])
  })

  it('boundary check detects forbidden literals in sample input', () => {
    expect(findForbiddenLiterals("const id = 'md.numen.sample'", ['md.numen.sample'])).toHaveLength(
      1,
    )
    expect(
      findForbiddenLiterals("const kind = 'reader:reader-pdf'", ['md.numen.sample']),
    ).toHaveLength(1)
    expect(findForbiddenLiterals("const clean = 'note'", ['md.numen.sample'])).toHaveLength(0)
  })
})

describe('plugin frontend loading', () => {
  it('loads the frontend of every plugin declaring one', async () => {
    const frontendPlugins = readdirSync(pluginsDir).filter((name) => {
      const frontendDir = join(pluginsDir, name, 'frontend')
      try {
        return statSync(frontendDir).isDirectory()
      } catch {
        return false
      }
    })

    expect(frontendPlugins.length).toBeGreaterThan(0)
    for (const name of frontendPlugins) {
      const manifestPath = join(pluginsDir, name, 'manifest.json')
      const manifest = JSON.parse(readFileSync(manifestPath, 'utf8')) as { id: string }
      const loader = pluginCatalog.loaders[manifest.id]
      expect(loader, `missing loader for plugin ${name} (${manifest.id})`).toBeDefined()
      const loaded = (await loader!()) as {
        activate?: unknown
        default?: { activate?: unknown }
      }
      expect(
        typeof (loaded.activate ?? loaded.default?.activate),
        `plugin ${name} does not export activate function`,
      ).toBe('function')
    }
  })
})

describe('restoring a session with a disabled plugin', () => {
  it('restores a session holding a tab of a disabled plugin without throwing', async () => {
    const originalLocation = globalThis.location
    try {
      // @ts-expect-error mocking location for session restore
      delete globalThis.location
      globalThis.location = new URL(
        'http://localhost/?tab=disabled-plugin:/docs/sample.pdf',
      ) as unknown as Location

      const window = await mountWindow()
      expect(getTabs(window)).toEqual([
        { id: 'disabled-plugin:/docs/sample.pdf', title: 'Plugin unavailable' },
      ])
      expect(window.text()).toContain('Plugin unavailable')
      expect(window.text()).toContain('disabled-plugin')
    } finally {
      globalThis.location = originalLocation
    }
  })

  it('restores a disabled plugin tab in windowTabs directly without throwing', async () => {
    const window = useWindowTabs()
    const id = await window.createTab('disabled-plugin', '/docs/sample.pdf')

    expect(id).toBe('disabled-plugin:/docs/sample.pdf')
    expect(window.tabs.value).toEqual([{ id, title: 'Plugin unavailable' }])
  })
})
