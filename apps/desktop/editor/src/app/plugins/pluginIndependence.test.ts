/**
 * Frontend architectural boundary tests that keep the host and plugins independent.
 */
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it, vi } from 'vitest'
import { useWindowTabs } from '@/entities/tab'
import { getTabs, mountWindow } from '@/testing/window'

const editorSrcDir = resolve(import.meta.dirname, '../..')

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

function findForbiddenLiterals(content: string): string[] {
  const found: string[] = []
  const idMatch = content.match(pluginIdRegex)
  if (idMatch) {
    found.push(`spells out plugin id ${idMatch[0]}`)
  }
  const readerMatch = content.match(readerKindRegex)
  if (readerMatch) {
    found.push(`spells out reader tab kind ${readerMatch[0]}`)
  }
  return found
}

describe('host and plugin independence', () => {
  it('host sources name no plugin id or plugin tab kind', () => {
    const hostSources = listHostSources(editorSrcDir)
    expect(hostSources.length).toBeGreaterThan(10)

    const violations: { file: string; issues: string[] }[] = []

    for (const source of hostSources) {
      const content = readFileSync(source, 'utf8')
      const issues = findForbiddenLiterals(content)
      if (issues.length > 0) {
        violations.push({ file: source.replace(editorSrcDir, 'src'), issues })
      }
    }

    expect(violations).toStrictEqual([])
  })

  it('boundary check detects forbidden literals in sample input', () => {
    expect(findForbiddenLiterals("const id = 'md.numen.sample'")).toHaveLength(1)
    expect(findForbiddenLiterals("const kind = 'reader:reader-pdf'")).toHaveLength(1)
    expect(findForbiddenLiterals("const clean = 'note'")).toHaveLength(0)
  })
})

describe('restoring a session with a disabled plugin', () => {
  it('restores a session holding a tab of a disabled plugin without throwing', async () => {
    try {
      vi.stubGlobal(
        'location',
        new URL(
          'http://localhost/?tab=disabled-plugin:/docs/sample.pdf',
        ),
      )

      const window = await mountWindow()
      expect(getTabs(window)).toEqual([
        { id: 'disabled-plugin:/docs/sample.pdf', title: 'sample.pdf' },
      ])
      expect(window.text()).toContain('Plugin disabled')
      expect(window.text()).toContain('disabled-plugin')
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('restores a disabled plugin tab in windowTabs directly without throwing', async () => {
    const window = useWindowTabs()
    const id = await window.createTab('disabled-plugin', '/docs/sample.pdf')

    expect(id).toBe('disabled-plugin:/docs/sample.pdf')
    expect(window.tabs.value).toEqual([{ id, title: 'sample.pdf' }])
  })
})
