import { describe, expect, it } from 'vitest'
import type { WindowTab } from '@/shared/model/tabKinds'
import { formatDetachedTab, parseInitialTab } from './detachedTab'

const createMockTab = (kind: string, path?: string): WindowTab =>
  ({
    kind: {
      kind,
      ...(path !== undefined ? { getOpenTab: () => ({ path }) } : {}),
    },
    state: {},
  }) as unknown as WindowTab

describe('formatDetachedTab', () => {
  it('returns the kind and file path when the tab holds an open path', () => {
    const getTab = () => createMockTab('plex', 'physics/Ontology.md')

    expect(formatDetachedTab(getTab, 'plex:248cab16-f508-4fb4-a820-16b6f9090fc3')).toBe(
      'plex:physics/Ontology.md',
    )
  })

  it('returns only the kind when the tab holds no open path', () => {
    const getTab = () => createMockTab('plex', '')

    expect(formatDetachedTab(getTab, 'plex:248cab16-f508-4fb4-a820-16b6f9090fc3')).toBe('plex')
  })

  it('returns only the kind when the tab kind does not report an open path', () => {
    const getTab = () => createMockTab('settings')

    expect(formatDetachedTab(getTab, 'settings:settings')).toBe('settings')
  })

  it('falls back to the raw tab id when the tab is not found', () => {
    expect(formatDetachedTab(() => null, 'note:hello.md')).toBe('note:hello.md')
  })
})

describe('parseInitialTab', () => {
  it('parses a bare kind with no target', () => {
    expect(parseInitialTab('plex')).toStrictEqual({ kind: 'plex', target: '' })
  })

  it('parses a kind and vault path target', () => {
    expect(parseInitialTab('note:hello.md')).toStrictEqual({ kind: 'note', target: 'hello.md' })
    expect(parseInitialTab('plex:physics/Ontology.md')).toStrictEqual({
      kind: 'plex',
      target: 'physics/Ontology.md',
    })
  })

  it('strips generated UUID suffixes so they are not treated as vault paths', () => {
    expect(parseInitialTab('plex:248cab16-f508-4fb4-a820-16b6f9090fc3')).toStrictEqual({
      kind: 'plex',
      target: '',
    })
  })

  it('strips repeated kind identities such as settings:settings', () => {
    expect(parseInitialTab('settings:settings')).toStrictEqual({
      kind: 'settings',
      target: '',
    })
  })

  it('preserves the reader prefix when parsing reader tab kinds', () => {
    expect(parseInitialTab('reader:epub:books/guide.epub')).toStrictEqual({
      kind: 'reader:epub',
      target: 'books/guide.epub',
    })
    expect(parseInitialTab('reader:epub')).toStrictEqual({
      kind: 'reader:epub',
      target: '',
    })
  })
})
