/** Whether the host runs a command of a plugin over a note: its when condition. Every plugin here is invented. */
import { describe, expect, it } from 'vitest'
import { isCommandAvailable } from './commandWhen'

const declaration = (when?: string) => ({
  pluginId: 'org.example.ledger',
  id: 'ledger.sum',
  title: 'Sum',
  ...(when === undefined ? {} : { when }),
})

const NOTE = { path: 'notes/Bees.md', title: 'Bees', type: 'link' }
const NONE = { path: '', title: '', type: 'note' }
const deps = {
  getContentTypes: () => ({ md: 'text/markdown' }),
  getWindowValues: () => ({ 'window.isVaultReady': true, config: { ledger: { isOn: true } } }),
}

describe('a command of a plugin', () => {
  it('runs over anything without a condition', () => {
    expect(isCommandAvailable(declaration(), NONE, deps)).toBe(true)
  })

  it('over a note runs only where a note is given, of the type its condition names', () => {
    expect(isCommandAvailable(declaration('target.isNote'), NOTE, deps)).toBe(true)
    expect(isCommandAvailable(declaration('target.isNote'), NONE, deps)).toBe(false)
    expect(isCommandAvailable(declaration("target.noteType == 'link'"), NOTE, deps)).toBe(true)
    expect(
      isCommandAvailable(declaration("target.contentType == 'text/markdown'"), NOTE, deps),
    ).toBe(true)
  })

  it('reads the window and the settings', () => {
    expect(
      isCommandAvailable(
        declaration('window.isVaultReady && config.ledger.isOn == true'),
        NONE,
        deps,
      ),
    ).toBe(true)
    expect(isCommandAvailable(declaration('window.isTabOpen'), NONE, deps)).toBe(false)
  })

  it('whose condition cannot be read does not run', () => {
    expect(isCommandAvailable(declaration('target.isRemote'), NOTE, deps)).toBe(false)
  })
})
