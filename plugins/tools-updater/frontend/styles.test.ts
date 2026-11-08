/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const css = readFileSync(resolve(import.meta.dirname, 'styles.css'), 'utf8')

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

describe('the popover of the update badge', () => {
  it('opens below its trigger, as the title bar holds the trigger', () => {
    const rule = readRule('.update-popover')

    expect(rule.get('position')).toBe('absolute')
    expect(rule.get('top')).toBe('calc(100% + 4px)')
    expect(rule.has('bottom')).toBe(false)
  })
})


describe('the chip of the version', () => {
  it('is flat, 20px high, 12px in text, 8px in padding, and still', () => {
    const rule = readRule('.update-indicator__trigger')

    expect(rule.get('--chip-height')).toBe('20px')
    expect(rule.get('--chip-text')).toBe('12px')
    expect(rule.get('--chip-pad')).toBe('8px')
    expect(rule.get('border')).toBe('none')
    expect(rule.get('border-radius')).toBe('var(--numen-radius)')
    expect(css).not.toMatch(/animation|@keyframes|transition|focus-visible|outline:\s*var/)
  })
})
