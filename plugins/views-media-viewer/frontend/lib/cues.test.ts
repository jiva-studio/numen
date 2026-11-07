/**
 * Transcript words mapped back to the millisecond positions they were recorded
 * at.
 *
 * The application rejects a transcript whose cues run backwards, overlap, or end
 * before they begin, so every result is checked for that as well as for the
 * computed times.
 */
import { describe, expect, it } from 'vitest'
import { applyCues, findCueAt, isSame, spanCues, getText, type Cue } from './cues'

const CUES: readonly Cue[] = [
  { text: 'A bell over the door.', from: 1_000, to: 3_000 },
  { text: 'Rain on the awning.', from: 3_000, to: 6_000 },
  { text: 'Someone counting change.', from: 6_000, to: 10_000 },
]

/** Cues run forward, none begins before the one before it ends. */
const isOrdered = (cues: readonly Cue[]) => {
  for (let index = 0; index < cues.length; index++) {
    const one = cues[index]!
    expect(one.from).toBeLessThanOrEqual(one.to)
    expect(Number.isInteger(one.from)).toBe(true)
    expect(Number.isInteger(one.to)).toBe(true)
    if (index > 0) expect(one.from).toBeGreaterThanOrEqual(cues[index - 1]!.to)
  }
  return cues
}

/** applyCues, with the result checked for cue order. */
const applyText = (was: readonly Cue[], text: string) => isOrdered(applyCues(was, text))

describe('the words as they were left', () => {
  it('are the cues they came from', () => {
    expect(applyText(CUES, getText(CUES))).toStrictEqual(CUES)
  })

  it('are one line to a cue', () => {
    expect(getText(CUES)).toBe(
      'A bell over the door.\nRain on the awning.\nSomeone counting change.',
    )
  })
})

describe('a line whose words changed', () => {
  it('keeps the times of an edited line', () => {
    const result = applyText(
      CUES,
      'A bell above the door.\nRain on the awning.\nSomeone counting change.',
    )

    expect(result).toStrictEqual([
      { text: 'A bell above the door.', from: 1_000, to: 3_000 },
      CUES[1],
      CUES[2],
    ])
  })

  it('keeps the times of an edited line in any position', () => {
    const result = applyText(
      CUES,
      'A bell over the door.\nRain on the roof.\nSomeone counting change.',
    )

    expect(result[1]).toStrictEqual({ text: 'Rain on the roof.', from: 3_000, to: 6_000 })
  })

  it('keeps every one of them where they all changed', () => {
    const result = applyText(CUES, 'One.\nTwo.\nThree.')

    expect(result).toStrictEqual([
      { text: 'One.', from: 1_000, to: 3_000 },
      { text: 'Two.', from: 3_000, to: 6_000 },
      { text: 'Three.', from: 6_000, to: 10_000 },
    ])
  })
})

describe('lines joined into one', () => {
  it('run from the first of them to the last, where two are joined', () => {
    const result = applyText(
      CUES,
      'A bell over the door. Rain on the awning.\nSomeone counting change.',
    )

    expect(result).toStrictEqual([
      { text: 'A bell over the door. Rain on the awning.', from: 1_000, to: 6_000 },
      CUES[2],
    ])
  })

  it('run from the first to the last, where three are joined', () => {
    const result = applyText(
      CUES,
      'A bell over the door. Rain on the awning. Someone counting change.',
    )

    expect(result).toStrictEqual([
      {
        text: 'A bell over the door. Rain on the awning. Someone counting change.',
        from: 1_000,
        to: 10_000,
      },
    ])
  })

  it('run from the first to the last where the joined words were changed too', () => {
    const result = applyText(CUES, 'A bell, and rain.\nSomeone counting change.')

    expect(result[0]).toStrictEqual({ text: 'A bell, and rain.', from: 1_000, to: 6_000 })
  })
})

describe('a line split in two', () => {
  it('divides its span where the split fell in its characters', () => {
    // Ten characters against ten, over a span of two seconds.
    const was: readonly Cue[] = [{ text: 'abcdeABCDE', from: 2_000, to: 4_000 }]

    expect(applyText(was, 'abcde\nABCDE')).toStrictEqual([
      { text: 'abcde', from: 2_000, to: 3_000 },
      { text: 'ABCDE', from: 3_000, to: 4_000 },
    ])
  })

  it('divides it in proportion where the split fell off centre', () => {
    const was: readonly Cue[] = [{ text: 'abcdefgh', from: 0, to: 8_000 }]

    expect(applyText(was, 'ab\ncdefgh')).toStrictEqual([
      { text: 'ab', from: 0, to: 2_000 },
      { text: 'cdefgh', from: 2_000, to: 8_000 },
    ])
  })

  it('gives the whole span to the second where the split fell at the very start', () => {
    const was: readonly Cue[] = [{ text: 'abcdefgh', from: 1_000, to: 9_000 }]

    expect(applyText(was, '\nabcdefgh')).toStrictEqual([
      { text: 'abcdefgh', from: 1_000, to: 9_000 },
    ])
  })

  it('gives the whole span to the first where the split fell at the very end', () => {
    const was: readonly Cue[] = [{ text: 'abcdefgh', from: 1_000, to: 9_000 }]

    expect(applyText(was, 'abcdefgh\n')).toStrictEqual([
      { text: 'abcdefgh', from: 1_000, to: 9_000 },
    ])
  })

  it('divides the span of the line it fell in and leaves the others alone', () => {
    const result = applyText(
      CUES,
      'A bell over the door.\nRain on\n the awning.\nSomeone counting change.',
    )

    expect(result[0]).toStrictEqual(CUES[0])
    expect(result[1]!.from).toBe(3_000)
    expect(result[2]!.to).toBe(6_000)
    expect(result[1]!.to).toBe(result[2]!.from)
    expect(result[3]).toStrictEqual(CUES[2])
  })

  it('divides it in three where two splits fell in it', () => {
    const was: readonly Cue[] = [{ text: 'aabbcc', from: 0, to: 3_000 }]

    expect(applyText(was, 'aa\nbb\ncc')).toStrictEqual([
      { text: 'aa', from: 0, to: 1_000 },
      { text: 'bb', from: 1_000, to: 2_000 },
      { text: 'cc', from: 2_000, to: 3_000 },
    ])
  })
})

describe('a line emptied entirely', () => {
  it('is dropped, and the lines around it are left alone', () => {
    expect(applyText(CUES, 'A bell over the door.\n\nSomeone counting change.')).toStrictEqual([
      CUES[0],
      CUES[2],
    ])
  })

  it('is dropped where it is the last line', () => {
    expect(applyText(CUES, 'A bell over the door.\nRain on the awning.\n')).toStrictEqual([
      CUES[0],
      CUES[1],
    ])
  })

  it('is dropped where it is the first line', () => {
    expect(applyText(CUES, '\nRain on the awning.\nSomeone counting change.')).toStrictEqual([
      CUES[1],
      CUES[2],
    ])
  })
})

describe('a transcript deleted entirely', () => {
  it('holds no cues at all', () => {
    expect(applyText(CUES, '')).toStrictEqual([])
  })

  it('drops a cue whose line is deleted', () => {
    expect(applyText([CUES[0]!], '')).toStrictEqual([])
  })
})

describe('a line typed where no cue stood', () => {
  it('takes an empty span at the end of the words before it', () => {
    const result = applyText(CUES, `${getText(CUES)}\nAnd the door again.`)

    expect(result[3]).toStrictEqual({ text: 'And the door again.', from: 10_000, to: 10_000 })
  })

  it('takes an empty span at the start of the words after it', () => {
    const result = applyText(CUES, `A first thought.\n${getText(CUES)}`)

    expect(result[0]).toStrictEqual({ text: 'A first thought.', from: 1_000, to: 1_000 })
  })

  it('takes an empty span between the words either side of it', () => {
    const result = applyText(
      CUES,
      'A bell over the door.\nA thought.\nRain on the awning.\nSomeone counting change.',
    )

    expect(result[1]).toStrictEqual({ text: 'A thought.', from: 3_000, to: 3_000 })
  })

  it('returns the typed lines for a recording with no cues', () => {
    expect(applyText([], 'A thought.')).toStrictEqual([{ text: 'A thought.', from: 0, to: 0 }])
  })
})

describe('a cue that spans no time at all', () => {
  it('gives both halves of a split cue its single moment', () => {
    const was: readonly Cue[] = [{ text: 'abcd', from: 5_000, to: 5_000 }]

    expect(applyText(was, 'ab\ncd')).toStrictEqual([
      { text: 'ab', from: 5_000, to: 5_000 },
      { text: 'cd', from: 5_000, to: 5_000 },
    ])
  })

  it('is joined to the one before it without moving either', () => {
    const was: readonly Cue[] = [
      { text: 'A bell.', from: 1_000, to: 3_000 },
      { text: 'And rain.', from: 3_000, to: 3_000 },
    ]

    expect(applyText(was, 'A bell. And rain.')).toStrictEqual([
      { text: 'A bell. And rain.', from: 1_000, to: 3_000 },
    ])
  })
})

describe('a line of nothing but spaces', () => {
  it('is a cue like any other, and is not dropped', () => {
    const result = applyText(CUES, 'A bell over the door.\n   \nSomeone counting change.')

    expect(result).toStrictEqual([CUES[0], { text: '   ', from: 3_000, to: 6_000 }, CUES[2]])
  })
})

describe('lines that read areNeighbourhoodsEqual', () => {
  it('leave the one that stayed where it was, and drop the other', () => {
    const was: readonly Cue[] = [
      { text: 'Again.', from: 0, to: 1_000 },
      { text: 'Again.', from: 1_000, to: 2_000 },
    ]

    expect(applyText(was, 'Again.')).toStrictEqual([{ text: 'Again.', from: 0, to: 1_000 }])
  })
})

describe('the words as the application gave them', () => {
  it('are given back unchanged, so a transcript nobody edited is not written', () => {
    expect(isSame(applyCues(CUES, getText(CUES)), CUES)).toBe(true)
  })

  it('are given back unchanged where the editor added a newline of its own', () => {
    expect(isSame(applyCues(CUES, `${getText(CUES)}\n`), CUES)).toBe(true)
  })

  it('are not what a changed word gives back, however small the change', () => {
    const text = getText(CUES).replace('awning', 'awnings')

    expect(isSame(applyCues(CUES, text), CUES)).toBe(false)
  })

  it('are not what a line moved past another gives back', () => {
    const text = [CUES[1]!.text, CUES[0]!.text, CUES[2]!.text].join('\n')

    expect(isSame(applyCues(CUES, text), CUES)).toBe(false)
  })

  it('are not what the same words at another moment give back', () => {
    const moved = CUES.map((cue) => ({ ...cue, from: cue.from + 1 }))

    expect(isSame(moved, CUES)).toBe(false)
  })
})

describe('every line on screen', () => {
  it('is answered for, an emptied one included', () => {
    const text = 'A bell over the door.\n\nSomeone counting change.'

    expect(spanCues(CUES, text).length).toBe(text.split('\n').length)
    expect(spanCues(CUES, text)[1]).toStrictEqual({ text: '', from: 3_000, to: 6_000 })
  })

  it('is answered for where the transcript was emptied', () => {
    expect(spanCues(CUES, '').length).toBe(1)
  })
})

describe('findCueAt', () => {
  it('is the one whose span holds it', () => {
    expect(findCueAt(CUES, 1_000)).toBe(0)
    expect(findCueAt(CUES, 4_500)).toBe(1)
    expect(findCueAt(CUES, 6_000)).toBe(2)
  })

  it('returns the previous cue during a silence', () => {
    expect(findCueAt(CUES, 10_500)).toBe(2)
  })

  it('is nothing before the first cue begins', () => {
    expect(findCueAt(CUES, 0)).toBe(-1)
    expect(findCueAt([], 4_000)).toBe(-1)
  })
})

describe('cues that differ only in how many there are', () => {
  it('are not the same words, whichever run is the shorter', () => {
    expect(isSame(CUES.slice(0, 2), CUES)).toBe(false)
    expect(isSame(CUES, CUES.slice(0, 2))).toBe(false)
  })
})

describe('cues that differ only in where one ends', () => {
  it('are not the same words', () => {
    const later = CUES.map((cue, index) => (index === 1 ? { ...cue, to: cue.to + 1 } : cue))

    expect(isSame(CUES, later)).toBe(false)
  })
})

describe('lines typed past the cue that stood there', () => {
  it('are all given an empty span after the cue they repeat', () => {
    const was: readonly Cue[] = [{ text: 'A bell.', from: 1_000, to: 3_000 }]

    expect(spanCues(was, 'A bell.\nA bell.\nA bell.')).toStrictEqual([
      { text: 'A bell.', from: 1_000, to: 3_000 },
      { text: 'A bell.', from: 3_000, to: 3_000 },
      { text: 'A bell.', from: 3_000, to: 3_000 },
    ])
  })
})

describe('empty lines laid over one span', () => {
  it('share its start, and the last of them takes its end', () => {
    const was: readonly Cue[] = [{ text: 'A bell.', from: 1_000, to: 3_000 }]

    expect(spanCues(was, '\n')).toStrictEqual([
      { text: '', from: 1_000, to: 1_000 },
      { text: '', from: 1_000, to: 3_000 },
    ])
  })
})
