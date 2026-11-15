import { describe, expect, it } from 'vitest'
import {
  findActionStart,
  findNextStart,
  findPreviousStart,
  findStepAt,
  getBadgeAt,
  getCursorAt,
  getHighlightsAt,
  getSegments,
  getSetupLength,
  getStepPhases,
  getVideoLength,
  type Input,
} from './playback.js'

const starts = [0.5, 2, null, 4.25]

const fromSource = <T extends (...args: never[]) => unknown>(fn: T): T =>
  new Function(`return ${fn.toString()}`)() as T

describe('findStepAt', () => {
  it('finds the last step that started by the time', () => {
    expect(findStepAt(starts, 0.5)).toBe(0)
    expect(findStepAt(starts, 3.9)).toBe(1)
    expect(findStepAt(starts, 60)).toBe(3)
  })

  it('finds none before the first step', () => {
    expect(findStepAt(starts, 0.2)).toBe(-1)
  })
})

describe('findNextStart', () => {
  it('finds the start of the next step that ran', () => {
    expect(findNextStart(starts, 0)).toBe(0.5)
    expect(findNextStart(starts, 2.5)).toBe(4.25)
  })

  it('moves on from a step the video stands at', () => {
    expect(findNextStart(starts, 1.99)).toBe(4.25)
    expect(findNextStart(starts, 2.01)).toBe(4.25)
  })

  it('finds none after the last step', () => {
    expect(findNextStart(starts, 4.3)).toBeNull()
  })
})

describe('findPreviousStart', () => {
  it('goes back to the start of the current step after a second of it', () => {
    expect(findPreviousStart(starts, 3.5)).toBe(2)
  })

  it('goes to the step before within the first second of a step', () => {
    expect(findPreviousStart(starts, 4.5)).toBe(2)
    expect(findPreviousStart(starts, 2.2)).toBe(0.5)
  })

  it('goes to the start of the video from the first step', () => {
    expect(findPreviousStart(starts, 0.8)).toBe(0)
    expect(findPreviousStart(starts, 0.1)).toBe(0)
  })
})

describe('getStepPhases', () => {
  it('marks the steps before the current one played and the ones after it to come', () => {
    expect(getStepPhases(4, 1)).toEqual(['done', 'current', 'ahead', 'ahead'])
  })

  it('marks every step to come before the first step', () => {
    expect(getStepPhases(2, -1)).toEqual(['ahead', 'ahead'])
  })
})

describe('getSegments', () => {
  it('spans each step that ran from its start to the next start or the end', () => {
    expect(getSegments(starts, 5)).toEqual([
      { index: 0, left: 10, width: 30 },
      { index: 1, left: 40, width: 45 },
      { index: 3, left: 85, width: 15 },
    ])
  })

  it('keeps the last of the steps that start together', () => {
    expect(getSegments([0, 0, 0, 1], 2)).toEqual([
      { index: 2, left: 0, width: 50 },
      { index: 3, left: 50, width: 50 },
    ])
  })

  it('gives no segments for a video of unknown length', () => {
    expect(getSegments(starts, 0)).toEqual([])
  })
})

describe('getVideoLength', () => {
  const steps = [
    { start: 0.5, duration: 1200 },
    { start: 2, duration: 3000 },
    { start: null, duration: 0 },
  ]

  it('takes the duration of the media', () => {
    expect(getVideoLength(7.5, steps)).toBe(7.5)
  })

  it('takes the end of the last step when the media has no duration', () => {
    expect(getVideoLength(Number.POSITIVE_INFINITY, steps)).toBe(5)
    expect(getVideoLength(Number.NaN, steps)).toBe(5)
  })
})

describe('getSetupLength', () => {
  const steps = (...keywords: string[]) => keywords.map((keyword) => ({ keyword }))

  it('counts the Background and Given steps with the And after them', () => {
    expect(getSetupLength(steps('Given', 'And', 'Given', 'But', 'When', 'Then'))).toBe(4)
  })

  it('stops at a Then that comes before any When', () => {
    expect(getSetupLength(steps('Given', 'Then', 'When'))).toBe(1)
  })

  it('counts every step of a scenario with no When or Then', () => {
    expect(getSetupLength(steps('Given', 'And'))).toBe(2)
  })

  it('counts none for a scenario that starts with an action', () => {
    expect(getSetupLength(steps('When', 'And', 'Then'))).toBe(0)
  })

  it('counts none for no steps', () => {
    expect(getSetupLength([])).toBe(0)
  })
})

describe('findActionStart', () => {
  it('finds the start of the first step after the setup that ran', () => {
    expect(findActionStart(starts, 1)).toBe(2)
    expect(findActionStart(starts, 2)).toBe(4.25)
  })

  it('finds none when no step after the setup ran', () => {
    expect(findActionStart([1, 1, null], 2)).toBeNull()
    expect(findActionStart([1, 1], 2)).toBeNull()
  })
})

describe('the functions as the page runs them', () => {
  it('run from their own source', () => {
    expect(fromSource(findStepAt)(starts, 2.5)).toBe(1)
    expect(fromSource(findNextStart)(starts, 2.5)).toBe(4.25)
    expect(fromSource(findPreviousStart)(starts, 3.5)).toBe(2)
    expect(fromSource(getStepPhases)(2, 0)).toEqual(['current', 'ahead'])
    expect(fromSource(getSegments)(starts, 5)).toHaveLength(3)
    expect(fromSource(getVideoLength)(Number.NaN, [{ start: 1, duration: 500 }])).toBe(1.5)
    expect(fromSource(getSetupLength)([{ keyword: 'Given' }, { keyword: 'When' }])).toBe(1)
    expect(fromSource(findActionStart)(starts, 1)).toBe(2)
  })
})

const createInput = (kind: Input['kind'], t: number, fields: Partial<Input> = {}): Input => ({
  kind,
  t,
  x: 0,
  y: 0,
  ...fields,
})

const createClick = (t: number, x: number, y: number): Input[] => [
  createInput('down', t, { x, y }),
  createInput('up', t + 0.05, { x, y }),
]

const createKey = (t: number, key: string, fields: Partial<Input> = {}): Input =>
  createInput('key', t, { key, mods: [], x: 0.1, y: 0.2, w: 0.3, h: 0.04, ...fields })

const createTyping = (t: number, text: string, fields: Partial<Input> = {}): Input =>
  createInput('type', t, { text, x: 0.4, y: 0.1, w: 0.2, h: 0.05, ...fields })

describe('getCursorAt', () => {
  const clicks = [...createClick(2, 0.2, 0.4), ...createClick(5, 0.6, 0.8)]

  it('shows no cursor in a video without pointer input', () => {
    expect(getCursorAt([], 3)).toBeNull()
    expect(getCursorAt([createKey(1, 'Enter')], 3)).toBeNull()
  })

  it('starts at the centre of the frame and travels to the first click', () => {
    expect(getCursorAt(clicks, 0.5)).toMatchObject({ x: 0.5, y: 0.5, isPressed: false })
    const halfway = getCursorAt(clicks, 1.7)
    expect(halfway?.x).toBeCloseTo(0.35)
    expect(halfway?.y).toBeCloseTo(0.45)
    expect(getCursorAt(clicks, 2)).toMatchObject({ x: 0.2, y: 0.4 })
  })

  it('starts a ripple on the point of a click and ends it after 0.4 s', () => {
    expect(getCursorAt(clicks, 2)?.ripple).toEqual({ x: 0.2, y: 0.4, progress: 0 })
    expect(getCursorAt(clicks, 2.2)?.ripple?.progress).toBeCloseTo(0.5)
    expect(getCursorAt(clicks, 2.41)?.ripple).toBeNull()
  })

  it('shows the button pressed between down and up', () => {
    expect(getCursorAt(clicks, 2.02)?.isPressed).toBe(true)
    expect(getCursorAt(clicks, 2.1)?.isPressed).toBe(false)
  })

  it('rests on a click until 0.6 s before the next one', () => {
    expect(getCursorAt(clicks, 4.4)).toMatchObject({ x: 0.2, y: 0.4 })
  })

  it('travels to the next click with easing and lands on it', () => {
    const halfway = getCursorAt(clicks, 4.7)
    expect(halfway?.x).toBeCloseTo(0.4)
    expect(halfway?.y).toBeCloseTo(0.6)
    expect(getCursorAt(clicks, 4.55)?.x).toBeLessThan(0.3)
    expect(getCursorAt(clicks, 5)).toMatchObject({ x: 0.6, y: 0.8 })
  })

  it('travels between clicks closer than 0.6 s in the time between them', () => {
    const quick = [createInput('down', 1, { x: 0, y: 0 }), createInput('down', 1.2, { x: 1, y: 1 })]
    expect(getCursorAt(quick, 1.1)?.x).toBeCloseTo(0.5)
  })

  it('follows the recorded path of a drag at its own pace, pressed', () => {
    const drag = [
      createInput('down', 1, { x: 0.1, y: 0.1 }),
      createInput('move', 1.03, { x: 0.2, y: 0.1 }),
      createInput('move', 1.06, { x: 0.2, y: 0.4 }),
      createInput('up', 1.09, { x: 0.2, y: 0.4 }),
    ]
    expect(getCursorAt(drag, 1.015)).toMatchObject({ isPressed: true })
    expect(getCursorAt(drag, 1.015)?.x).toBeCloseTo(0.15)
    expect(getCursorAt(drag, 1.045)?.y).toBeCloseTo(0.25)
    expect(getCursorAt(drag, 1.2)).toMatchObject({ x: 0.2, y: 0.4, isPressed: false })
  })

  it('stays on the last click after it', () => {
    expect(getCursorAt(clicks, 9)).toEqual({ x: 0.6, y: 0.8, ripple: null, isPressed: false })
  })

  it('works from its source text, as the report page runs it', () => {
    expect(fromSource(getCursorAt)(clicks, 4.7)?.x).toBeCloseTo(0.4)
  })
})

describe('getBadgeAt', () => {
  it('names a key with its modifiers', () => {
    expect(getBadgeAt([createKey(1, 'k', { mods: ['Meta'] })], 1.1)?.parts).toEqual(['⌘ K'])
    expect(getBadgeAt([createKey(1, 'Tab', { mods: ['Shift', 'Control'] })], 1.1)?.parts).toEqual([
      '⌃ ⇧ Tab',
    ])
    expect(getBadgeAt([createKey(1, 'Escape')], 1.1)?.parts).toEqual(['Esc'])
    expect(getBadgeAt([createKey(1, 'ArrowDown')], 1.1)?.parts).toEqual(['↓'])
  })

  it('groups keys pressed within 0.6 s of each other into one badge', () => {
    const keys = [createKey(1, 'ArrowDown'), createKey(1.5, 'ArrowDown'), createKey(2, 'Enter')]
    expect(getBadgeAt(keys, 2.1)?.parts).toEqual(['↓', '↓', 'Enter'])
    expect(getBadgeAt(keys, 1.2)?.parts).toEqual(['↓'])
  })

  it('starts a new badge after a pause', () => {
    const keys = [createKey(1, 'Escape'), createKey(1.7, 'Enter')]
    expect(getBadgeAt(keys, 1.8)?.parts).toEqual(['Enter'])
  })

  it('fades a badge out within a second of the last key', () => {
    expect(getBadgeAt([createKey(1, 'Enter')], 1.5)?.opacity).toBe(1)
    expect(getBadgeAt([createKey(1, 'Enter')], 1.85)?.opacity).toBeCloseTo(0.5)
    expect(getBadgeAt([createKey(1, 'Enter')], 2)).toBeNull()
    expect(getBadgeAt([createKey(1, 'Enter')], 0.5)).toBeNull()
  })

  it('shows the value a field holds after the typing', () => {
    expect(getBadgeAt([createTyping(1, 'Shopping list')], 1.2)?.parts).toEqual([
      'Typing “Shopping list”',
    ])
  })

  it('collapses the letters of typing into one badge', () => {
    const typing = [
      createKey(1, 'a', { isField: true }),
      createTyping(1, 'a'),
      createKey(1.05, 'B', { isField: true, mods: ['Shift'] }),
      createTyping(1.05, 'aB'),
      createKey(1.1, 'Enter', { isField: true }),
    ]
    expect(getBadgeAt(typing, 1.2)?.parts).toEqual(['Typing “aB”', 'Enter'])
  })

  it('joins the text typed into an editable element', () => {
    const typing = [
      createTyping(1, 'Milk', { isAppend: true }),
      createTyping(1.1, ' and tea', { isAppend: true }),
      createTyping(1.2, '', { isAppend: true }),
    ]
    expect(getBadgeAt(typing, 1.3)?.parts).toEqual(['Typing “Milk and tea”'])
  })

  it('shows a letter pressed outside a field as a key', () => {
    expect(getBadgeAt([createKey(1, 'j')], 1.1)?.parts).toEqual(['J'])
  })

  it('shows the last 40 characters of long typing', () => {
    const text = 'a'.repeat(30) + 'b'.repeat(20)
    const part = getBadgeAt([createTyping(1, text)], 1.1)?.parts[0]
    expect(part).toBe(`Typing “…${'a'.repeat(19)}${'b'.repeat(20)}”`)
  })

  it('shows the dots recorded for a password', () => {
    expect(getBadgeAt([createTyping(1, '••••')], 1.1)?.parts).toEqual(['Typing “••••”'])
  })

  it('works from its source text, as the report page runs it', () => {
    expect(fromSource(getBadgeAt)([createKey(1, 'Enter')], 1.1)?.parts).toEqual(['Enter'])
  })
})

describe('getHighlightsAt', () => {
  it('rings the element a key went to for 1 s, fading out over the last 0.3 s', () => {
    const keys = [createKey(1, 'Enter')]
    expect(getHighlightsAt(keys, 1)).toEqual([
      { x: 0.1, y: 0.2, w: 0.3, h: 0.04, opacity: 1, index: 0 },
    ])
    expect(getHighlightsAt(keys, 1.5)[0]?.opacity).toBe(1)
    expect(getHighlightsAt(keys, 1.85)[0]?.opacity).toBeCloseTo(0.5)
    expect(getHighlightsAt(keys, 2)).toEqual([])
  })

  it('fades the old ring out over 0.15 s once a new element is ringed', () => {
    const keys = [createKey(1, 'Enter'), createTyping(1.2, 'Milk')]
    expect(getHighlightsAt(keys, 1.2).map((ring) => ring.index)).toEqual([0, 1])
    expect(getHighlightsAt(keys, 1.275)[0]?.opacity).toBeCloseTo(0.5)
    expect(getHighlightsAt(keys, 1.35).map((ring) => ring.index)).toEqual([1])
  })

  it('keeps one ring while the same element takes more keys', () => {
    const keys = [
      createTyping(1, 'M'),
      createTyping(1.1, 'Mi'),
      createKey(1.2, 'Enter', { x: 0.4, y: 0.1, w: 0.2, h: 0.05 }),
    ]
    expect(getHighlightsAt(keys, 1.9)).toEqual([
      { x: 0.4, y: 0.1, w: 0.2, h: 0.05, opacity: 1, index: 2 },
    ])
  })

  it('rings nothing for a key without a focused element or for a click', () => {
    expect(getHighlightsAt([createKey(1, 'Escape', { w: 0, h: 0 })], 1.1)).toEqual([])
    expect(getHighlightsAt(createClick(1, 0.5, 0.5), 1.1)).toEqual([])
  })

  it('works from its source text, as the report page runs it', () => {
    expect(fromSource(getHighlightsAt)([createKey(1, 'Enter')], 1.1)[0]?.x).toBe(0.1)
  })
})
