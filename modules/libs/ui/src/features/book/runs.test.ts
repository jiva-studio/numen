/**
 * The runs of a drawn document read off the markup.
 *
 * The offsets are bytes of the book's text and the drawn nodes are strings, so
 * every case here is set in Cyrillic and Devanagari: a letter is two bytes and
 * three, and no offset below is a count of characters.
 */
import { describe, expect, it } from 'vitest'
import { marksIn, offsetAt, rangesOver, runAt, runsIn } from './runs'

/** A document as it is drawn, held in an element the way the reader holds it. */
const paperOf = (markup: string): HTMLElement => {
  const paper = document.createElement('div')
  paper.innerHTML = markup
  document.body.append(paper)
  return paper
}

/** Two runs standing against each other, in a script counted in two bytes. */
const TWO = '<p><span data-offset="0">Один</span><span data-offset="8">два</span></p>'

describe('the runs of a drawn document', () => {
  it('are the elements carrying an offset, in the order the markup sets them', () => {
    const runs = runsIn(paperOf(TWO))

    expect(runs.map((run) => run.at)).toStrictEqual([0, 8])
    expect(runs[1]?.element.textContent).toBe('два')
  })

  it('leave out an element carrying no offset anybody can read', () => {
    const runs = runsIn(paperOf('<p data-offset="कुछ">x</p><p data-offset="4">y</p>'))

    expect(runs.map((run) => run.at)).toStrictEqual([4])
  })

  it('are none at all in a document with no text in it', () => {
    expect(runsIn(paperOf('<nav><a href="one.xhtml">One</a></nav>'))).toStrictEqual([])
  })
})

describe('the run an offset falls in', () => {
  const runs = runsIn(paperOf(TWO))

  it('is the last one beginning at or before it', () => {
    expect(runAt(runs, 0)?.at).toBe(0)
    expect(runAt(runs, 7)?.at).toBe(0)
    expect(runAt(runs, 8)?.at).toBe(8)
    expect(runAt(runs, 4_000)?.at).toBe(8)
  })

  it('is no run at all before the first of them', () => {
    expect(runAt(runs, -1)).toBeUndefined()
  })
})

describe('a place named inside the document', () => {
  it('stands at the offset of the run holding the name', () => {
    const paper = paperOf('<p><span data-offset="6">प्रथम<b id="here">x</b></span></p>')

    expect(offsetAt(paper, runsIn(paper), 'here')).toBe(6)
  })

  it('stands at the first run inside a name holding several', () => {
    const paper = paperOf('<section id="parva"><p data-offset="40">आदि</p></section>')

    expect(offsetAt(paper, runsIn(paper), 'parva')).toBe(40)
  })

  it('stands at the first run after a name standing between runs', () => {
    // A book names its chapters on empty anchors, and the place the name points
    // at is where its text begins.
    const paper = paperOf(
      '<p data-offset="0">Первая</p><h2 id="second"></h2><p data-offset="12">Вторая</p>',
    )

    expect(offsetAt(paper, runsIn(paper), 'second')).toBe(12)
  })

  it('stands nowhere where the document names no such place', () => {
    const paper = paperOf(TWO)

    expect(offsetAt(paper, runsIn(paper), 'nowhere')).toBeUndefined()
  })

  it('stands nowhere where nothing was named', () => {
    const paper = paperOf(TWO)

    expect(offsetAt(paper, runsIn(paper), '')).toBeUndefined()
  })

  it('stands nowhere where the name comes after every run of the document', () => {
    const paper = paperOf('<p data-offset="0">Первая</p><h2 id="ends"></h2>')

    expect(offsetAt(paper, runsIn(paper), 'ends')).toBeUndefined()
  })
})

describe('a stretch of the book marked where it stands', () => {
  it('runs over the nodes the markup carries, counted in bytes of them', () => {
    const paper = paperOf(TWO)

    const [range] = rangesOver(runsIn(paper), [{ begins: 0, ends: 8 }])

    expect(range?.toString()).toBe('Один')
  })

  it('opens and closes inside the runs, and not at their edges', () => {
    // Two bytes a letter, so four bytes into the first run is two letters in,
    // and the stretch closes one letter into the second.
    const paper = paperOf(TWO)

    const [range] = rangesOver(runsIn(paper), [{ begins: 4, ends: 10 }])

    expect(range?.toString()).toBe('инд')
  })

  it('is left out where it reaches past the text the document holds', () => {
    const paper = paperOf(TWO)

    expect(rangesOver(runsIn(paper), [{ begins: 100, ends: 110 }])).toStrictEqual([])
  })

  it('is left out where it begins before the first run', () => {
    const paper = paperOf('<p data-offset="900">Вторая глава</p>')

    expect(rangesOver(runsIn(paper), [{ begins: 10, ends: 20 }])).toStrictEqual([])
  })

  it('is nothing at all in a document with no runs in it', () => {
    expect(rangesOver([], [{ begins: 0, ends: 8 }])).toStrictEqual([])
  })
})

/** A run the browser has put somewhere, which jsdom never does by itself. */
const standing = (markup: string, lefts: readonly number[]): HTMLElement => {
  const paper = paperOf(markup)
  const runs = paper.querySelectorAll<HTMLElement>('[data-offset]')
  runs.forEach((run, at) => {
    if (at >= lefts.length) return
    Object.defineProperty(run, 'getClientRects', {
      value: () => [{ left: lefts[at] }],
      configurable: true,
    })
  })
  return paper
}

describe('where the runs stand across the columns', () => {
  it('is where the first of the rectangles a run has begins', () => {
    const paper = standing(TWO, [120, 480])

    const marks = marksIn(runsIn(paper), 40)

    expect(marks).toStrictEqual([
      { at: 0, x: 80 },
      { at: 8, x: 440 },
    ])
  })

  it('is nothing for a run the browser has put nowhere', () => {
    // A run broken over a column edge has a rectangle in each column, and a
    // document nothing has laid out has no rectangles at all.
    const paper = standing(TWO, [120])

    expect(marksIn(runsIn(paper), 0)).toStrictEqual([{ at: 0, x: 120 }])
  })
})
