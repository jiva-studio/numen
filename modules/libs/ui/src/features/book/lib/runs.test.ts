/**
 * The runs of a drawn document read off the markup.
 *
 * The offsets are bytes of the book's text and the drawn nodes are strings, so
 * every case here is set in Cyrillic and Devanagari: a letter is two bytes and
 * three, and no offset below is a count of characters.
 */
import { describe, expect, it } from 'vitest'
import { createMarks, findOffset, rangesOver, findRun, getRuns } from './runs'

/** A document as it is drawn, held in an element the way the reader holds it. */
const createMarkupPaper = (markup: string): HTMLElement => {
  const paper = document.createElement('div')
  paper.innerHTML = markup
  document.body.append(paper)
  return paper
}

/** Two runs standing against each other, in a script counted in two bytes. */
const TWO = '<p><span data-offset="0">Один</span><span data-offset="8">два</span></p>'

describe('the runs of a drawn document', () => {
  it('are the elements carrying an offset, in the order the markup sets them', () => {
    const runs = getRuns(createMarkupPaper(TWO))

    expect(runs.map((run) => run.at)).toStrictEqual([0, 8])
    expect(runs[1]?.element.textContent).toBe('два')
  })

  it('leave out an element carrying no offset anybody can read', () => {
    const runs = getRuns(createMarkupPaper('<p data-offset="कुछ">x</p><p data-offset="4">y</p>'))

    expect(runs.map((run) => run.at)).toStrictEqual([4])
  })

  it('are none at all in a document with no text in it', () => {
    expect(getRuns(createMarkupPaper('<nav><a href="one.xhtml">One</a></nav>'))).toStrictEqual([])
  })
})

describe('the run an offset falls in', () => {
  const runs = getRuns(createMarkupPaper(TWO))

  it('is the last one beginning at or before it', () => {
    expect(findRun(runs, 0)?.at).toBe(0)
    expect(findRun(runs, 7)?.at).toBe(0)
    expect(findRun(runs, 8)?.at).toBe(8)
    expect(findRun(runs, 4_000)?.at).toBe(8)
  })

  it('is no run at all before the first of them', () => {
    expect(findRun(runs, -1)).toBeUndefined()
  })
})

describe('a place named inside the document', () => {
  it('remains at the offset of the run holding the name', () => {
    const paper = createMarkupPaper('<p><span data-offset="6">प्रथम<b id="here">x</b></span></p>')

    expect(findOffset(paper, getRuns(paper), 'here')).toBe(6)
  })

  it('remains at the first run inside a name holding several', () => {
    const paper = createMarkupPaper('<section id="parva"><p data-offset="40">आदि</p></section>')

    expect(findOffset(paper, getRuns(paper), 'parva')).toBe(40)
  })

  it('remains at the first run after a name standing between runs', () => {
    // A book names its chapters on empty anchors, and the place the name points
    // at is where its text begins.
    const paper = createMarkupPaper(
      '<p data-offset="0">Первая</p><h2 id="second"></h2><p data-offset="12">Вторая</p>',
    )

    expect(findOffset(paper, getRuns(paper), 'second')).toBe(12)
  })

  it('remains nowhere where the document names no such place', () => {
    const paper = createMarkupPaper(TWO)

    expect(findOffset(paper, getRuns(paper), 'nowhere')).toBeUndefined()
  })

  it('remains nowhere where nothing was named', () => {
    const paper = createMarkupPaper(TWO)

    expect(findOffset(paper, getRuns(paper), '')).toBeUndefined()
  })

  it('remains nowhere where the name arrives after every run of the document', () => {
    const paper = createMarkupPaper('<p data-offset="0">Первая</p><h2 id="ends"></h2>')

    expect(findOffset(paper, getRuns(paper), 'ends')).toBeUndefined()
  })
})

describe('a stretch of the book marked at its current position', () => {
  it('runs over the nodes the markup carries, counted in bytes of them', () => {
    const paper = createMarkupPaper(TWO)

    const [range] = rangesOver(getRuns(paper), [{ from: 0, to: 8 }])

    expect(range?.toString()).toBe('Один')
  })

  it('opens and closes inside the runs, and not at their edges', () => {
    // Two bytes a letter, so four bytes into the first run is two letters in,
    // and the stretch closes one letter into the second.
    const paper = createMarkupPaper(TWO)

    const [range] = rangesOver(getRuns(paper), [{ from: 4, to: 10 }])

    expect(range?.toString()).toBe('инд')
  })

  it('is left out where it reaches past the text the document holds', () => {
    const paper = createMarkupPaper(TWO)

    expect(rangesOver(getRuns(paper), [{ from: 100, to: 110 }])).toStrictEqual([])
  })

  it('is left out where it begins before the first run', () => {
    const paper = createMarkupPaper('<p data-offset="900">Вторая глава</p>')

    expect(rangesOver(getRuns(paper), [{ from: 10, to: 20 }])).toStrictEqual([])
  })

  it('is nothing at all in a document with no runs in it', () => {
    expect(rangesOver([], [{ from: 0, to: 8 }])).toStrictEqual([])
  })
})

/** A run the browser has put somewhere, which jsdom never does by itself. */
const createPaper = (markup: string, lefts: readonly number[]): HTMLElement => {
  const paper = createMarkupPaper(markup)
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
    const paper = createPaper(TWO, [120, 480])

    const marks = createMarks(getRuns(paper), 40)

    expect(marks).toStrictEqual([
      { at: 0, x: 80 },
      { at: 8, x: 440 },
    ])
  })

  it('is nothing for a run the browser has put nowhere', () => {
    // A run broken over a column edge has a rectangle in each column, and a
    // document nothing has laid out has no rectangles at all.
    const paper = createPaper(TWO, [120])

    expect(createMarks(getRuns(paper), 0)).toStrictEqual([{ at: 0, x: 120 }])
  })
})

describe('a place named by nothing', () => {
  it('remains nowhere even where an element carries an empty id', () => {
    const paper = createMarkupPaper('<p id="" data-offset="3">Первая</p>')

    expect(findOffset(paper, getRuns(paper), '')).toBeUndefined()
  })
})

describe('a stretch of the book over a run of several nodes', () => {
  const several = '<p data-offset="0"><span>Один</span><b>два</b></p>'

  it('begins in the node holding its first byte, counted from the run', () => {
    const paper = createMarkupPaper(several)

    const [range] = rangesOver(getRuns(paper), [{ from: 10, to: 12 }])

    expect(range?.toString()).toBe('в')
  })

  it('closes at the end of the node its last byte ends', () => {
    const paper = createMarkupPaper(several)

    const [range] = rangesOver(getRuns(paper), [{ from: 0, to: 8 }])

    expect(range?.endContainer.textContent).toBe('Один')
    expect(range?.endOffset).toBe(4)
  })

  it('begins at the end of the node its first byte is the end of', () => {
    const paper = createMarkupPaper(several)

    const [range] = rangesOver(getRuns(paper), [{ from: 8, to: 12 }])

    expect(range?.startContainer.textContent).toBe('Один')
    expect(range?.startOffset).toBe(4)
  })

  it('runs from inside a later run, counted from where that run begins', () => {
    const paper = createMarkupPaper(TWO)

    const [range] = rangesOver(getRuns(paper), [{ from: 10, to: 12 }])

    expect(range?.toString()).toBe('в')
  })

  it('is left out where it begins past the text of its run and ends in a later one', () => {
    const paper = createMarkupPaper('<p data-offset="0">Один</p><p data-offset="20">два</p>')

    expect(rangesOver(getRuns(paper), [{ from: 12, to: 22 }])).toStrictEqual([])
  })

  it('is left out where it ends before every run it could begin in', () => {
    const paper = createMarkupPaper('<p data-offset="5">Один</p>')

    expect(rangesOver(getRuns(paper), [{ from: 0, to: 8 }])).toStrictEqual([])
  })
})
