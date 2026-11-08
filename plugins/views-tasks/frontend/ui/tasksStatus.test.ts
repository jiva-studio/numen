/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const css = readFileSync(resolve(import.meta.dirname, 'tasksStatus.css'), 'utf8')

const readRule = (selector: string): Map<string, string> => {
  const start = css.indexOf(`${selector} {`)
  const body = css.slice(css.indexOf('{', start) + 1, css.indexOf('}', start))
  const declarations = body
    .split(';')
    .map((line) => line.replace(/\/\*[\s\S]*?\*\//g, '').trim())
    .filter((line) => line.includes(':'))
  return new Map(
    declarations.map((line) => {
      const colon = line.indexOf(':')
      return [line.slice(0, colon).trim(), line.slice(colon + 1).trim()]
    }),
  )
}

describe('the chip of the background tasks', () => {
  it('is flat, 20px high, 12px in text, 8px in padding, and still', () => {
    const rule = readRule('.tasks-status__trigger')

    expect(rule.get('--chip-height')).toBe('20px')
    expect(rule.get('--chip-text')).toBe('12px')
    expect(rule.get('--chip-pad')).toBe('8px')
    expect(rule.get('border')).toBe('none')
    expect(rule.get('border-radius')).toBe('var(--numen-radius)')
    expect(css).not.toMatch(/animation|@keyframes|focus-visible|outline:\s*var|__badge/)
  })
})
