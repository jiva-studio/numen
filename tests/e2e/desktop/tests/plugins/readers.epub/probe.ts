/**
 * A recorder of what the EPUB reader shows, sampled on every animation frame and on every mutation,
 * scroll and transition event of the reading area, so a turn that shows a page, then the page
 * before it, and then the new page again leaves that sequence behind.
 */
import type { Page } from 'playwright/test'

/** What the reader showed at one moment. */
export interface Sample {
  /** Milliseconds since the recording began. */
  readonly t: number
  /** What caused the sample: a frame, a mutation, a scroll or a transition event. */
  readonly source: string
  /** The footer counter, such as `3 of 40`. */
  readonly count: string
  /** The offset of the run at the top left of the reading area, and null over a gap or when hidden. */
  readonly offset: number | null
  /** How far the columns are moved sideways as drawn now, in CSS pixels. */
  readonly shift: number
  /** How far the columns are being moved to, in CSS pixels. */
  readonly target: number
  /** The offset the drawn chapter begins at. */
  readonly chapter: number
  /** Whether the columns are hidden until they stand at their spread. */
  readonly hidden: boolean
  /** How far the reading area itself is scrolled, which should stay zero. */
  readonly scrolled: number
  /** How wide the columns run in all, which changes only where the text is laid out again. */
  readonly width: number
  /** The size of the reading area, such as `900x600`. */
  readonly size: string
}

declare global {
  interface Window {
    bookProbe?: { samples: Sample[]; stop: () => void }
  }
}

/** Starts recording in the page. The reader is drawn in a plugin's open shadow root. */
export const startProbe = async (page: Page): Promise<void> => {
  await page.evaluate(() => {
    window.bookProbe?.stop()
    const findAll = (selector: string): Element[] => {
      const found: Element[] = []
      const visit = (root: Document | ShadowRoot) => {
        found.push(...root.querySelectorAll(selector))
        for (const element of root.querySelectorAll('*')) {
          if (element.shadowRoot) visit(element.shadowRoot)
        }
      }
      visit(document)
      return found
    }
    const area = findAll('.book__area')[0] as HTMLElement | undefined
    const count = findAll('.book__count')[0]
    const paper = area?.querySelector<HTMLElement>('.book__paper')
    if (!area || !count || !paper) throw new Error('no book on screen')
    const began = performance.now()
    const samples: Sample[] = []

    const read = (source: string): void => {
      const runs = Array.from(paper.querySelectorAll<HTMLElement>('[data-offset]'))
      const box = area.getBoundingClientRect()
      const [shift = '0'] = getComputedStyle(paper).translate.split(' ')
      const [target = '0'] = paper.style.translate.split(' ')
      const hidden = getComputedStyle(paper).visibility === 'hidden'
      const run = runs.find((one) => {
        const rect = one.getClientRects()[0]
        return !!rect && rect.right > box.left + 1 && rect.left < box.right && rect.bottom > box.top
      })
      samples.push({
        t: Math.round(performance.now() - began),
        source,
        count: count.textContent ?? '',
        offset: run && !hidden ? Number(run.dataset['offset']) : null,
        shift: Number.parseFloat(shift) || 0,
        target: Number.parseFloat(target) || 0,
        chapter: Number(paper.querySelector<HTMLElement>('[data-offset]')?.dataset['offset'] ?? -1),
        hidden,
        scrolled: area.scrollLeft,
        width: paper.scrollWidth,
        size: `${area.clientWidth}x${area.clientHeight}`,
      })
    }

    let frame = 0
    const tick = () => {
      read('frame')
      frame = requestAnimationFrame(tick)
    }
    const observer = new MutationObserver(() => read('mutation'))
    observer.observe(paper, { attributes: true, childList: true })
    observer.observe(count, { childList: true, characterData: true, subtree: true })
    const onScroll = () => read('scroll')
    const onTransition = (event: Event) => read(event.type)
    area.addEventListener('scroll', onScroll)
    const names = ['transitionrun', 'transitionstart', 'transitionend', 'transitioncancel']
    for (const name of names) paper.addEventListener(name, onTransition)
    const onFonts = () => read('fonts')
    document.fonts.addEventListener('loadingdone', onFonts)
    const resizes = new ResizeObserver(() => read('resize'))
    resizes.observe(area)
    resizes.observe(paper)
    read('start')
    frame = requestAnimationFrame(tick)

    window.bookProbe = {
      samples,
      stop: () => {
        cancelAnimationFrame(frame)
        observer.disconnect()
        area.removeEventListener('scroll', onScroll)
        for (const name of names) paper.removeEventListener(name, onTransition)
        document.fonts.removeEventListener('loadingdone', onFonts)
        resizes.disconnect()
      },
    }
  })
}

/** Takes the samples recorded so far, and starts the next batch empty. */
export const takeSamples = async (page: Page): Promise<Sample[]> =>
  await page.evaluate(() => {
    const probe = window.bookProbe
    if (!probe) return []
    return probe.samples.splice(0, probe.samples.length)
  })

/** Which way the reader moves: forward is toward larger offsets. */
export type Direction = 'forward' | 'back'

const readPage = (count: string): number => Number(count.split(' of ')[0])

/** The values of a series with no two equal ones in a row. */
const collapse = <T>(values: readonly T[]): T[] =>
  values.filter((one, index) => index === 0 || one !== values[index - 1])

/** The offsets the reader showed, in order, each once. */
export const getShownOffsets = (samples: readonly Sample[]): number[] =>
  collapse(samples.flatMap((one) => (one.offset === null ? [] : [one.offset])))

/** The footer counts, in order, each once. */
export const getShownPages = (samples: readonly Sample[]): number[] =>
  collapse(samples.map((one) => readPage(one.count)))

/** The places the columns were sent to, in order, each once. */
export const getSentTargets = (samples: readonly Sample[]): number[] =>
  collapse(samples.map((one) => one.target))

/** The steps of a series that go against the direction, as `from>to`. */
const findReturns = (series: readonly number[], way: Direction, slack = 0): string[] => {
  const returns: string[] = []
  for (let index = 1; index < series.length; index += 1) {
    const step = (series[index] ?? 0) - (series[index - 1] ?? 0)
    if (way === 'forward' ? step < -slack : step > slack) {
      returns.push(`${series[index - 1]}>${series[index]}`)
    }
  }
  return returns
}

/**
 * Every place where what the reader showed went back toward the page before: the offsets at the
 * top left, the counter, the target of the columns and the scroll of the area.
 */
export const findReturnsOf = (samples: readonly Sample[], way: Direction): string[] => {
  const offsets = findOffsetReturns(samples, way)
  const pages = findReturns(getShownPages(samples), way).map((one) => `count ${one}`)
  const scrolls = samples
    .filter((one) => one.scrolled !== 0)
    .map((one) => `scrolled ${one.scrolled}`)
  const targets = findTargetReturns(samples, way)
  const layout = findLayoutChanges(samples)
  return [...offsets, ...pages, ...scrolls, ...targets, ...layout]
}

/** A short text of what was shown, for a failure message and a report. */
export const describeSamples = (samples: readonly Sample[]): string =>
  `offsets ${getShownOffsets(samples).join(' ')} | counts ${getShownPages(samples).join(' ')} | targets ${getSentTargets(samples).join(' ')}`

/**
 * The places where the columns of one chapter were sent back toward the page before. The columns
 * of a chapter are carried by a negative shift, which grows in size going forward, and a new
 * chapter begins again at zero.
 */
const findTargetReturns = (samples: readonly Sample[], way: Direction): string[] => {
  const shown = samples.filter((one) => !one.hidden)
  const chapters = new Set(shown.map((one) => one.chapter))
  return [...chapters].flatMap((chapter) => {
    const own = shown.filter((one) => one.chapter === chapter)
    const sizes = getSentTargets(own).map((one) => -one)
    return findReturns(sizes, way, 1).map((one) => `target ${one} in chapter ${chapter}`)
  })
}

/**
 * Where the text moved under columns that stood still, as a picture or a face arriving does, or the
 * reading area took another size. A turn changes neither.
 */
const findLayoutChanges = (samples: readonly Sample[]): string[] => {
  const shown = samples.filter((one) => !one.hidden)
  const found: string[] = []
  for (let index = 1; index < shown.length; index += 1) {
    const [before, now] = [shown[index - 1], shown[index]]
    if (!before || !now || before.chapter !== now.chapter) continue
    if (
      isStanding(before) &&
      isStanding(now) &&
      before.target === now.target &&
      before.offset !== now.offset
    ) {
      found.push(
        `drift ${before.offset}>${now.offset} at ${now.t} on ${now.source} with columns ${before.width}>${now.width}`,
      )
    }
    if (before.size !== now.size)
      found.push(`size ${before.size}>${now.size} at ${now.t} on ${now.source}`)
  }
  return found
}

/** The shift of the columns as drawn on each frame, each value once, for a report. */
export const getShiftSeries = (samples: readonly Sample[]): string =>
  collapse(samples.map((one) => Math.round(one.shift))).join(' ')

/** Whether the columns stand still at their spread in a sample, with something to read. */
const isStanding = (sample: Sample): boolean =>
  sample.offset !== null && Math.abs(sample.shift - sample.target) < 0.5

/** The steps where the offset at the top left went back, each with the two samples around it. */
const findOffsetReturns = (samples: readonly Sample[], way: Direction): string[] => {
  const shown = samples.filter((one) => one.offset !== null)
  const found: string[] = []
  for (let index = 1; index < shown.length; index += 1) {
    const [before, now] = [shown[index - 1], shown[index]]
    if (!before || !now || before.chapter !== now.chapter) continue
    const step = (now.offset ?? 0) - (before.offset ?? 0)
    if (way === 'forward' ? step < 0 : step > 0) {
      found.push(`offset ${before.offset}>${now.offset}: ${JSON.stringify([before, now])}`)
    }
  }
  return found
}
