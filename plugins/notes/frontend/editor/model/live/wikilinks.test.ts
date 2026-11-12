import { describe, expect, it } from 'vitest'
import { getMarks } from './marks'
import { renderBracketMatch } from '../../lib/live/wikilinks'
import { createState } from '../../fixtures/state'

interface Drawn {
  readonly from: number
  readonly to: number
  readonly name: string
}

const getDrawnWikilinks = (doc: string, caret: number): Drawn[] => {
  const state = createState(doc, caret)
  const found: Drawn[] = []
  getMarks(state, 0, doc.length).between(0, doc.length, (from, to, deco) => {
    found.push({ from, to, name: (deco.spec as { class?: string }).class ?? 'hidden' })
  })
  return found.sort((a, b) => a.from - b.from || b.to - a.to)
}

const ELSEWHERE = '\n\nelsewhere'

describe('a wikilink with the caret away from it', () => {
  it('hides both brackets and links only the target', () => {
    const doc = `See [[Entropy]].${ELSEWHERE}`
    expect(getDrawnWikilinks(doc, doc.length)).toStrictEqual([
      { from: 4, to: 6, name: 'hidden' },
      { from: 6, to: 13, name: 'cm-wikilink' },
      { from: 13, to: 15, name: 'hidden' },
    ])
  })

  it('shows only the alias, and hides the brackets and the target before it', () => {
    const doc = `[[Entropy|disorder]]${ELSEWHERE}`
    expect(getDrawnWikilinks(doc, doc.length)).toStrictEqual([
      { from: 0, to: 10, name: 'hidden' },
      { from: 10, to: 18, name: 'cm-wikilink' },
      { from: 18, to: 20, name: 'hidden' },
    ])
  })

  it('shows the heading with the target', () => {
    const doc = `[[Entropy#Heat]]${ELSEWHERE}`
    expect(getDrawnWikilinks(doc, doc.length)).toStrictEqual([
      { from: 0, to: 2, name: 'hidden' },
      { from: 2, to: 14, name: 'cm-wikilink' },
      { from: 14, to: 16, name: 'hidden' },
    ])
  })

  it('draws two links of one line apart, one at the start and one at the end', () => {
    const doc = `[[A]] and [[B]]${ELSEWHERE}`
    const links = getDrawnWikilinks(doc, doc.length).filter((one) => one.name === 'cm-wikilink')
    expect(links).toStrictEqual([
      { from: 2, to: 3, name: 'cm-wikilink' },
      { from: 12, to: 13, name: 'cm-wikilink' },
    ])
  })

  it('leaves an unfinished link as plain text', () => {
    const doc = `[[b${ELSEWHERE}`
    expect(getDrawnWikilinks(doc, doc.length)).toStrictEqual([])
  })

  it('leaves a link inside code as plain text', () => {
    const doc = `\`[[A]]\`${ELSEWHERE}`
    expect(getDrawnWikilinks(doc, doc.length).filter((one) => one.name.includes('wikilink'))).toStrictEqual(
      [],
    )
  })

  it('draws nothing of the single brackets the markdown parser reads inside the link', () => {
    const doc = `[[Entropy]]${ELSEWHERE}`
    expect(getDrawnWikilinks(doc, doc.length).some((one) => one.name === 'cm-link')).toBe(false)
  })
})

describe('a wikilink with the caret in it or touching it', () => {
  it.each([0, 5, 11])(
    'shows both brackets muted and the whole link as one link, caret at %i',
    (caret) => {
      const doc = `[[Entropy]]${ELSEWHERE}`
      expect(getDrawnWikilinks(doc, caret)).toStrictEqual([
        { from: 0, to: 11, name: 'cm-wikilink' },
        { from: 0, to: 2, name: 'cm-wikilink-mark' },
        { from: 9, to: 11, name: 'cm-wikilink-mark' },
      ])
    },
  )

  it('mutes the target of an alias with the brackets', () => {
    const doc = `[[Entropy|disorder]]${ELSEWHERE}`
    expect(getDrawnWikilinks(doc, 3).map((one) => [one.from, one.to, one.name])).toStrictEqual([
      [0, 20, 'cm-wikilink'],
      [0, 2, 'cm-wikilink-mark'],
      [2, 10, 'cm-wikilink-mark'],
      [18, 20, 'cm-wikilink-mark'],
    ])
  })
})

describe('the highlight of a matching bracket', () => {
  it('leaves out the brackets of a wikilink', () => {
    const state = createState('[[Entropy]]', 0)
    const match = { matched: true, start: { from: 0, to: 1 }, end: { from: 10, to: 11 } }
    expect(renderBracketMatch(match, state)).toStrictEqual([])
  })

  it('marks the brackets of the text around it', () => {
    const state = createState('(a [b])', 0)
    const match = { matched: true, start: { from: 0, to: 1 }, end: { from: 6, to: 7 } }
    expect(renderBracketMatch(match, state)).toHaveLength(2)
  })
})
