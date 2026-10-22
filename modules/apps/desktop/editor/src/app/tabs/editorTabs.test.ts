/**
 * Which tab edits the file at a path.
 */
import { describe, expect, it } from 'vitest'
import { createNoteLookup, findEditorTabAt } from './editorTabs'

type WindowTabs = Parameters<typeof findEditorTabAt>[0]

const createWindowTabs = (
  tabs: Record<string, { editsFiles: boolean; path: string } | undefined>,
): WindowTabs =>
  ({
    tabs: { value: Object.keys(tabs).map((id) => ({ id, title: id })) },
    getTab: (id: string) => {
      const tab = tabs[id]
      if (!tab) return null
      return {
        state: tab,
        kind: {
          getOpenTab: (state: { path: string }) => ({ path: state.path }),
          ...(tab.editsFiles ? { flushPending: () => Promise.resolve() } : {}),
        },
      }
    },
  }) as unknown as WindowTabs

describe('the tab that edits the file at a path', () => {
  it('is the tab of a kind that writes files out, standing at the path', () => {
    const windowTabs = createWindowTabs({
      'document:a': { editsFiles: false, path: 'a.md' },
      'deck:a': { editsFiles: true, path: 'a.md' },
    })

    expect(findEditorTabAt(windowTabs, 'a.md')).toBe('deck:a')
  })

  it('is none where only a tab that edits nothing remains at the path', () => {
    const windowTabs = createWindowTabs({ 'document:a': { editsFiles: false, path: 'a.md' } })

    expect(findEditorTabAt(windowTabs, 'a.md')).toBeNull()
  })

  it('is none where no tab remains at the path', () => {
    const windowTabs = createWindowTabs({ 'deck:a': { editsFiles: true, path: 'a.md' } })

    expect(findEditorTabAt(windowTabs, 'b.md')).toBeNull()
  })

  it('is none for a tab the window no longer holds', () => {
    const windowTabs = createWindowTabs({ 'deck:a': undefined })

    expect(findEditorTabAt(windowTabs, 'a.md')).toBeNull()
  })
})

describe('what the window knows of the note open at a path', () => {
  const create = (reachedTab: string | null) =>
    createNoteLookup(
      {
        reached: { getTabAt: () => reachedTab },
        getTitle: (id) => (id === 'note:a' ? 'Alpha' : ''),
      },
      createWindowTabs({ 'deck:a': { editsFiles: true, path: 'a.md' } }),
    )

  it('is the tab of a note editor, called what that editor calls it', () => {
    const lookup = create('note:a')

    expect(lookup.getTabAt('a.md')).toBe('note:a')
    expect(lookup.getTitle('a.md')).toBe('Alpha')
  })

  it('is the tab of a kind that edits files where no note editor remains at the path, called what the tab is', () => {
    const lookup = create(null)

    expect(lookup.getTabAt('a.md')).toBe('deck:a')
    expect(lookup.getTitle('a.md')).toBe('deck:a')
  })

  it('is no tab and no title where nothing remains at the path', () => {
    const lookup = create(null)

    expect(lookup.getTabAt('b.md')).toBeNull()
    expect(lookup.getTitle('b.md')).toBe('')
  })
})
