import { describe, expect, it } from 'vitest'
import { createTemplateItems } from './groupTemplates'

const ids = (paths: readonly string[], root = 'Templates') =>
  createTemplateItems(
    root,
    paths.map((path) => ({ path })),
  ).map((item) => item.id)

describe('the items of the template picker', () => {
  it('uses the vault path as the id and the file name without the extension as the title', () => {
    expect(createTemplateItems('Templates', [{ path: 'Templates/Book review.md' }])).toEqual([
      { id: 'Templates/Book review.md', title: 'Book review' },
    ])
  })

  it('places a template of the root in no group', () => {
    const [item] = createTemplateItems('Templates', [{ path: 'Templates/Meeting.md' }])

    expect(item).toBeDefined()
    expect('group' in (item ?? {})).toBe(false)
  })

  it('places a template of a subfolder in the group of its path', () => {
    const items = createTemplateItems('Templates', [
      { path: 'Templates/Meetings/Weekly.md' },
      { path: 'Templates/Meetings/Quarterly/Review.md' },
    ])

    expect(items.map((item) => item.group)).toEqual(['Meetings', 'Meetings/Quarterly'])
  })

  it('reads the group relative to a root that has several segments', () => {
    const items = createTemplateItems('Work/Models', [{ path: 'Work/Models/Meetings/Weekly.md' }])

    expect(items).toEqual([
      { id: 'Work/Models/Meetings/Weekly.md', title: 'Weekly', group: 'Meetings' },
    ])
  })

  it('orders root templates before groups, and titles inside a group', () => {
    expect(
      ids([
        'Templates/zeta.md',
        'Templates/Weekly/b.md',
        'Templates/Alpha.md',
        'Templates/Weekly/A.md',
        'Templates/Daily/Note.md',
        'Templates/archive/x.md',
      ]),
    ).toEqual([
      'Templates/Alpha.md',
      'Templates/zeta.md',
      'Templates/archive/x.md',
      'Templates/Daily/Note.md',
      'Templates/Weekly/A.md',
      'Templates/Weekly/b.md',
    ])
  })

  it('orders templates of equal titles by path', () => {
    expect(ids(['Templates/a.md', 'Templates/A.md'])).toEqual(['Templates/A.md', 'Templates/a.md'])
    expect(ids(['Templates/a.md', 'Templates/A.md'].reverse())).toEqual([
      'Templates/A.md',
      'Templates/a.md',
    ])
  })

  it('returns no items for no templates', () => {
    expect(createTemplateItems('Templates', [])).toEqual([])
  })
})
