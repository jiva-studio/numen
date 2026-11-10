/**
 * Which commands a run names the file for: those whose condition reads the target.
 */
import { describe, expect, it } from 'vitest'
import { createWhenReader } from '@numen/plugin-loader'
import { getRunPath, isOverTarget } from './commandRun'

describe('a command over a target', () => {
  it('is one whose condition reads the target', () => {
    const reader = createWhenReader()

    expect(isOverTarget({ when: "target.contentType == 'application/pdf'" }, reader)).toBe(true)
    expect(isOverTarget({ when: 'window.isVaultReady' }, reader)).toBe(false)
    expect(isOverTarget({}, reader)).toBe(false)
  })

  it('is sent the file where it offers items or a place named the file', () => {
    const reader = createWhenReader()
    const run = { path: 'Shells.md' }

    expect(getRunPath({ hasItems: true }, run, reader)).toBe('Shells.md')
    expect(getRunPath({}, { ...run, isTargetNamed: true }, reader)).toBe('Shells.md')
    expect(getRunPath({}, run, reader)).toBe('')
  })
})
