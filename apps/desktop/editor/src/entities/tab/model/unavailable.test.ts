import { describe, expect, it, vi } from 'vitest'
import { createMockCatalogEntry } from '@numen/core-client/testing'
import { answerTabKinds } from '@/testing/tabCatalog'
import { NO_TAB_KIND_PLUGINS } from '@/testing/tabs'
import UnavailablePane from '../ui/UnavailablePane.vue'
import { createUnavailableKind, type UnavailableState } from './unavailable'

const waitForAnswer = (state: unknown) =>
  vi.waitFor(() => expect((state as UnavailableState).lookup.value.status).not.toBe('pending'))

describe('createUnavailableKind', () => {
  it('keeps the name of the missing kind and what the tab stood on', () => {
    const kind = createUnavailableKind(
      UnavailablePane,
      'reader:viewer',
      undefined,
      NO_TAB_KIND_PLUGINS,
    )

    const state = kind.open('/a.bin', 'tab')

    expect(kind.kind).toBe('reader:viewer')
    expect(kind.address?.getOpenTab?.(state)).toEqual({ path: '/a.bin' })
    expect(kind.address?.identity?.('/a.bin')).toBe('/a.bin')
  })

  it('keeps the state the plugin saved for the tab', () => {
    const kind = createUnavailableKind(
      UnavailablePane,
      'reader:viewer',
      undefined,
      NO_TAB_KIND_PLUGINS,
    )

    const state = kind.open('a.bin', 'tab', { page: 4 })

    expect(kind.address?.getState?.(state)).toEqual({ page: 4 })
    expect(kind.getTitle?.(state)).toBe('a.bin')
  })

  it('keeps the original tab title when provided', () => {
    const icon = { name: 'FilesIcon' }
    const kind = createUnavailableKind(
      UnavailablePane,
      'files',
      { title: 'Files', icon },
      NO_TAB_KIND_PLUGINS,
    )

    expect(kind.display?.icon).toBe(icon)
    expect(kind.getTitle?.(kind.open('', 'tab'))).toBe('Files')
  })

  it('draws the tab with the pane it is given', () => {
    expect(
      createUnavailableKind(UnavailablePane, 'files', undefined, NO_TAB_KIND_PLUGINS).pane,
    ).toBe(UnavailablePane)
  })

  it('names a tab with no path and no title after the kind the catalog declares once it answers', async () => {
    const { lookups, findTabKindPlugin } = answerTabKinds([
      createMockCatalogEntry('sample.decks', {
        tabKinds: [
          { kind: 'other', title: 'Other' },
          { kind: 'deck', title: 'Deck' },
        ],
      }),
    ])
    const kind = createUnavailableKind(UnavailablePane, 'deck', undefined, { findTabKindPlugin })
    const state = kind.open('', 'tab')
    expect(kind.getTitle?.(state)).toBe('deck')

    await waitForAnswer(state)

    expect(kind.getTitle?.(state)).toBe('Deck')
    expect(lookups).toStrictEqual(['deck'])
  })

  it('names a tab after its kind where the catalog declares no title for it', async () => {
    const { findTabKindPlugin } = answerTabKinds([
      createMockCatalogEntry('sample.bare', { tabKinds: ['bare'] }),
    ])
    const kind = createUnavailableKind(UnavailablePane, 'bare', undefined, { findTabKindPlugin })
    const state = kind.open('', 'tab')

    await waitForAnswer(state)

    expect(kind.getTitle?.(state)).toBe('bare')
  })

  it('names a tab after its kind where no plugin declares the kind', async () => {
    const { findTabKindPlugin } = answerTabKinds([
      createMockCatalogEntry('sample.other', { tabKinds: [{ kind: 'other', title: 'Other' }] }),
    ])
    const kind = createUnavailableKind(UnavailablePane, 'missing-kind', undefined, {
      findTabKindPlugin,
    })
    const state = kind.open('', 'tab')

    await waitForAnswer(state)

    expect(kind.getTitle?.(state)).toBe('missing-kind')
  })
})
