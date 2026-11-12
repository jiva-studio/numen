/** Whether the host runs a command of a plugin over a note: its when condition. Every plugin here is invented. */
import { describe, expect, it } from 'vitest'
import { createWhenReader } from '../when'
import { isCommandAvailable } from './commandWhen'

const declaration = (when?: string) => ({
  pluginId: 'org.example.ledger',
  id: 'ledger.sum',
  title: 'Sum',
  ...(when === undefined ? {} : { when }),
})

const TARGET = { isFolder: false, others: [], tab: '' }
const NOTE = { ...TARGET, path: 'notes/Bees.md', title: 'Bees', type: 'link' }
const NONE = { ...TARGET, path: '', title: '', type: 'note' }
const FOLDER = { ...TARGET, path: 'notes', title: 'notes', isFolder: true, type: 'note' }
const ROOT = { ...FOLDER, path: '', title: '' }
const ROWS = { ...NOTE, others: ['notes/Wasps.md'] }
const deps = {
  whenReader: createWhenReader(),
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

  it('over a folder runs over the folder of the target, the root folder too', () => {
    expect(isCommandAvailable(declaration('target.isFolder'), FOLDER, deps)).toBe(true)
    expect(isCommandAvailable(declaration('target.isFolder'), ROOT, deps)).toBe(true)
    expect(isCommandAvailable(declaration('target.isRoot'), ROOT, deps)).toBe(true)
    expect(isCommandAvailable(declaration('target.isFolder'), NOTE, deps)).toBe(false)
  })

  it('reads the other paths selected with the target', () => {
    expect(isCommandAvailable(declaration('!target.isMultiple'), ROWS, deps)).toBe(false)
    expect(isCommandAvailable(declaration('!target.isMultiple'), NOTE, deps)).toBe(true)
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
