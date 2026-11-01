// @vitest-environment node
/**
 * The corner radii the default theme renders with, and every radius token the
 * repository's components name.
 *
 * A `var()` naming an undeclared property without a fallback makes the whole
 * declaration compute to its initial value, which for `border-radius` is 0.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const ROOT = fileURLToPath(new URL('../../../../../../', import.meta.url))
const TOKENS = readFileSync(fileURLToPath(new URL('./tokens.css', import.meta.url)), 'utf8')
const SCANNED = ['apps', 'libs', 'plugins']
const SKIPPED = new Set(['node_modules', 'dist', 'build', 'built', 'gen', 'coverage', 'testdata'])
const STYLED = /\.(vue|css)$/

const DECLARED = /(--numen-radius[\w-]*)\s*:\s*([^;]+);/g
const NAMED = /var\(\s*(--numen-radius[\w-]*)\s*\)/g

/** The radius tokens `tokens.css` declares, by name, with their values. */
const getDeclaredRadii = (): Map<string, string> =>
  new Map([...TOKENS.matchAll(DECLARED)].map((found) => [found[1] ?? '', (found[2] ?? '').trim()]))

/** Every `.vue` and `.css` file under `dir`, outside build output and dependencies. */
function listStyledFiles(dir: string): string[] {
  const files: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name.startsWith('.') || SKIPPED.has(entry.name)) continue
    const path = join(dir, entry.name)
    if (entry.isDirectory()) files.push(...listStyledFiles(path))
    else if (STYLED.test(entry.name)) files.push(path)
  }
  return files
}

describe('the radius tokens', () => {
  it('are declared with a non-zero value in the default theme', () => {
    const radii = getDeclaredRadii()

    for (const name of [
      '--numen-radius',
      '--numen-radius-tight',
      '--numen-radius-panel',
      '--numen-radius-pill',
      '--numen-radius-field',
      '--numen-radius-bubble',
    ]) {
      expect(radii.get(name), name).toBeDefined()
      expect(radii.get(name), name).not.toMatch(/^0(\.0+)?(px|rem|em)?$/)
    }
  })

  it('cover every radius a component names without a fallback', () => {
    const radii = getDeclaredRadii()
    const undeclared: string[] = []
    for (const area of SCANNED) {
      for (const path of listStyledFiles(join(ROOT, area))) {
        for (const found of readFileSync(path, 'utf8').matchAll(NAMED)) {
          const name = found[1] ?? ''
          if (!radii.has(name)) undeclared.push(`${path.slice(ROOT.length)}: ${name}`)
        }
      }
    }
    expect(undeclared).toEqual([])
  })
})
