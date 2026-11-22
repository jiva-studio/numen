import { describe, expect, it } from 'vitest'
import { createTemplateItems } from './groupTemplates'

const found = (path: string, group = '') => ({ path, group })

const ids = (templates: readonly { path: string; group: string }[]) =>
  createTemplateItems(templates).map((item) => item.id)

describe('the items of the template picker', () => {
  it('uses the vault path as the id and the file name without the extension as the title', () => {
    expect(createTemplateItems([found('Templates/Book review.md')])).toEqual([
      { id: 'Templates/Book review.md', title: 'Book review' },
    ])
  })

  it('strips .template from the title', () => {
    expect(createTemplateItems([found('Notes/Weekly.template.md', 'Notes')])).toEqual([
      { id: 'Notes/Weekly.template.md', title: 'Weekly', group: 'Notes' },
    ])
  })

  it('keeps a title that is .template alone', () => {
    expect(createTemplateItems([found('Notes/.template.md', 'Notes')])).toEqual([
      { id: 'Notes/.template.md', title: '.template', group: 'Notes' },
    ])
  })

  it('places a template with no group in no group', () => {
    const [item] = createTemplateItems([found('Templates/Meeting.md')])

    expect(item).toBeDefined()
    expect('group' in (item ?? {})).toBe(false)
  })

  it('places a template in the group it was found in', () => {
    const items = createTemplateItems([
      found('Templates/Meetings/Weekly.md', 'Meetings'),
      found('Templates/Meetings/Quarterly/Review.md', 'Meetings/Quarterly'),
    ])

    expect(items.map((item) => item.group)).toEqual(['Meetings', 'Meetings/Quarterly'])
  })

  it('orders root templates before groups, and titles inside a group', () => {
    expect(
      ids([
        found('Templates/zeta.md'),
        found('Templates/Weekly/b.md', 'Weekly'),
        found('Templates/Alpha.md'),
        found('Templates/Weekly/A.md', 'Weekly'),
        found('Templates/Daily/Note.md', 'Daily'),
        found('Templates/archive/x.md', 'archive'),
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
    const a = [found('Templates/a.md'), found('Templates/A.md')]

    expect(ids(a)).toEqual(['Templates/A.md', 'Templates/a.md'])
    expect(ids([...a].reverse())).toEqual(['Templates/A.md', 'Templates/a.md'])
  })

  it('returns no items for no templates', () => {
    expect(createTemplateItems([])).toEqual([])
  })
})
