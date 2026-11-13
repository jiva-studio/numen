/** The when conditions of the plugins' commands over what is in front of the palette. Every plugin is invented. */
import { describe, expect, it } from 'vitest'
import { createWhenReader } from '@numen/plugin-loader'
import type { CommandTarget } from '@/shared/model/command'
import { createWhenAvailability, getWhenGroup, hasNoteVariable } from './whenContext'

const target = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab-1',
  kind: 'note',
  path: 'reef/Coral.md',
  title: 'Coral',
  file: 'reef/Coral.md',
  vault: { id: 'reef', name: 'Reef' },
  isReady: true,
  ...override,
})

describe('a condition of a command of a plugin', () => {
  const isAvailable = createWhenAvailability({
    getContentTypes: () => ({ md: 'text/markdown', pdf: 'application/pdf' }),
    getConfig: () => ({ org: { example: { ledger: { isOn: true } } } }),
    whenReader: createWhenReader(),
  })

  it('reads the note in front, its content type and the settings', () => {
    expect(isAvailable('target.isNote', target())).toBe(true)
    expect(
      isAvailable("target.contentType == 'application/pdf'", target({ path: '', file: 'a/b.pdf' })),
    ).toBe(true)
    expect(isAvailable('config.org.example.ledger.isOn == true', target())).toBe(true)
  })

  it('is false where it does not hold, and where it cannot be read', () => {
    expect(isAvailable('target.isNote', target({ path: '', file: '' }))).toBe(false)
    expect(isAvailable('target.isRemote', target())).toBe(false)
  })

  it('is true where there is none', () => {
    expect(isAvailable(undefined, target({ isReady: false }))).toBe(true)
  })
})

describe('a condition that reads the type or the properties of the note', () => {
  const notes: Record<string, { type: string; properties?: { media: string } }> = {
    'reef/Coral.md': { type: 'link', properties: { media: 'media/coral.mp4' } },
  }
  const isAvailable = createWhenAvailability({
    getContentTypes: () => ({}),
    getConfig: () => ({}),
    whenReader: createWhenReader(),
    getNote: (one) => notes[one.path],
  })
  const DELETE_MEDIA =
    "target.noteType == 'link' && has(target.properties.media) && target.properties.media != ''"

  it('holds over a note whose type and properties match, once the note is read', () => {
    expect(isAvailable("target.noteType == 'link'", target())).toBe(true)
    expect(isAvailable(DELETE_MEDIA, target())).toBe(true)
  })

  it('is false while the note is not read yet', () => {
    expect(isAvailable("target.noteType == 'link'", target({ path: 'reef/Kelp.md' }))).toBe(false)
  })

  it('does not ask for the note where its condition reads neither', () => {
    expect(isAvailable('target.isNote', target({ path: 'reef/Kelp.md' }))).toBe(true)
  })
})

describe('the group of a command in the palette', () => {
  const reader = createWhenReader()
  const getWhenGroupOf = (when: string | undefined) => getWhenGroup(reader.getCondition(when))
  const hasNoteVariableIn = (when: string) => hasNoteVariable(reader.getCondition(when))

  it('is note where its condition reads target.isNote, file where it reads another target value, window otherwise', () => {
    expect(getWhenGroupOf('target.isNote && window.isVaultReady')).toBe('note')
    expect(getWhenGroupOf("target.contentType == 'application/pdf'")).toBe('file')
    expect(getWhenGroupOf('window.isVaultOpen')).toBe('window')
    expect(getWhenGroupOf(undefined)).toBe('window')
  })

  it('reads target.isNote, or not', () => {
    expect(hasNoteVariableIn('!target.isNote')).toBe(true)
    expect(hasNoteVariableIn('target.isFile')).toBe(false)
  })
})
