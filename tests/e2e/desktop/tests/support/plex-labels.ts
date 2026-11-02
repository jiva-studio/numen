import type { Page } from 'playwright/test'

/** A point on the screen. */
export interface ScreenPoint {
  readonly x: number
  readonly y: number
}

/** The curve of a line sampled on the screen, and which way its samples run against the first note to the second. */
export interface SampledLine {
  readonly points: readonly ScreenPoint[]
  /** 1 where the samples run from the first note to the second, -1 where they run back. */
  readonly way: 1 | -1
}

/** Where the letters of one label are against the line, in pixels on the screen. */
export interface LabelGap {
  /**
   * How farthest the nearest edge of the letters is from the line, normal it: the ink from the top
   * of a capital to the bottom of a descender, measured from the baseline the browser set.
   * Positive when the letters are clear of the line, negative when the line runs through them.
   */
  readonly gap: number
  /** On which side of the line the letters are: positive on the right looking from the first note to the second. */
  readonly side: number
  /**
   * How far the middle of the ink of this very text is from the line, across it, in pixels, whichever
   * side it is on. The ink is read off pixels: the text is drawn in the font of the label into a
   * canvas, and the rows with ink in them give its top and bottom.
   */
  readonly centre: number
  /**
   * How far the middle of the lowercase letters is from the line, in the same way: the ink of an x.
   * The one label of a line is set so that this is nothing, whatever letters its words have.
   */
  readonly bandCentre: number
}

/** The rows with ink of a text, in pixels above and below its baseline. */
interface Ink {
  readonly above: number
  readonly below: number
}

/**
 * The ink of every label of the visible plex as the engine draws it: the text is drawn in the font
 * of the label into a canvas, eight times as large, and the rows with ink in them give its top
 * and bottom about the baseline. The ink of an x gives the lowercase band, whatever the words are.
 */
const readInks = (page: Page): Promise<Readonly<Record<string, { real: Ink; lower: Ink }>>> =>
  page.evaluate(() => {
    const GROW = 8
    const svg = [...document.querySelectorAll<SVGSVGElement>('svg.plex')].find(
      (one) => one.getBoundingClientRect().width > 0,
    )!
    const readInk = (font: string, size: number, text: string): Ink => {
      const canvas = document.createElement('canvas')
      const paint = canvas.getContext('2d', { willReadFrequently: true })!
      paint.font = font
      canvas.width = Math.ceil(GROW * (paint.measureText(text).width / size + 2) * size)
      canvas.height = GROW * 3 * size
      paint.font = font.replace(`${size}px`, `${GROW * size}px`)
      const baseline = GROW * 2 * size
      paint.fillText(text, GROW * size, baseline)
      const { data } = paint.getImageData(0, 0, canvas.width, canvas.height)
      const rows = Array.from({ length: canvas.height }, (_, row) => row).filter((row) =>
        data
          .slice(row * canvas.width * 4, (row + 1) * canvas.width * 4)
          .some((v, slot) => slot % 4 === 3 && v > 127),
      )
      return { above: (baseline - rows[0]!) / GROW, below: (rows.at(-1)! + 1 - baseline) / GROW }
    }
    const inks: Record<string, { real: Ink; lower: Ink }> = {}
    for (const letters of svg.querySelectorAll<SVGTextElement>('.plex__edge-label--letters')) {
      const style = getComputedStyle(letters)
      const size = Number.parseFloat(style.fontSize)
      const font = `${style.fontStyle} ${style.fontWeight} ${size}px ${style.fontFamily}`
      const words = letters.textContent?.trim() ?? ''
      inks[words] = { real: readInk(font, size, words), lower: readInk(font, size, 'x') }
    }
    return inks
  })

/**
 * Measures every label of the visible plex against one line, in the same way for a line with one
 * label and for a line with two: from the baseline the browser gave the first letter, so the
 * distance does not depend on the rotation of the text or on the engine's box for it.
 */
export const getLabelGaps = async (
  page: Page,
  line: SampledLine,
): Promise<Readonly<Record<string, LabelGap>>> => {
  const inks = await readInks(page)
  return page.evaluate(
    ({ points, way, inks }) => {
      const svg = [...document.querySelectorAll<SVGSVGElement>('svg.plex')].find(
        (one) => one.getBoundingClientRect().width > 0,
      )!
      const context = document.createElement('canvas').getContext('2d')!
      const gaps: Record<
        string,
        { gap: number; side: number; centre: number; bandCentre: number }
      > = {}
      for (const letters of svg.querySelectorAll<SVGTextElement>('.plex__edge-label--letters')) {
        const style = getComputedStyle(letters)
        context.font = `${style.fontStyle} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`
        const words = letters.textContent?.trim() ?? ''
        const ink = context.measureText('Hxg')
        const screen = letters.getScreenCTM()!
        const scale = Math.hypot(screen.a, screen.b)
        const origin = letters.getStartPositionOfChar(0).matrixTransform(screen)
        const turn = (letters.getRotationOfChar(0) * Math.PI) / 180
        // The way up the page of the letters, on the screen.
        const up = { x: Math.sin(turn), y: -Math.cos(turn) }

        let nearest = 0
        let best = Infinity
        points.forEach((point, step) => {
          const apart = Math.hypot(point.x - origin.x, point.y - origin.y)
          if (apart < best) {
            best = apart
            nearest = step
          }
        })
        const here = points[nearest]!
        const back = points[Math.max(nearest - 1, 0)]!
        const on = points[Math.min(nearest + 1, points.length - 1)]!
        const run = { x: (on.x - back.x) * way, y: (on.y - back.y) * way }
        const size = Math.hypot(run.x, run.y) || 1
        const normal = { x: -run.y / size, y: run.x / size }
        const lean = up.x * normal.x + up.y * normal.y
        // The baseline the browser gave the first letter, across the line.
        const base = (origin.x - here.x) * normal.x + (origin.y - here.y) * normal.y
        const top = base + lean * ink.actualBoundingBoxAscent * scale
        const bottom = base - lean * ink.actualBoundingBoxDescent * scale
        const closest = Math.min(top, bottom)
        const farthest = Math.max(top, bottom)
        const side = closest > 0 ? 1 : farthest < 0 ? -1 : 0
        const gap = side > 0 ? closest : side < 0 ? -farthest : -Math.min(farthest, -closest)
        const getCentre = (found: { above: number; below: number }) =>
          Math.abs(base + (lean * (found.above - found.below) * scale) / 2)
        gaps[words] = {
          gap,
          side,
          centre: getCentre(inks[words]!.real),
          bandCentre: getCentre(inks[words]!.lower),
        }
      }
      return gaps
    },
    { ...line, inks },
  )
}

/** The most a label may be from its line: the letters hug it. */
export const MAX_LABEL_GAP = 4

/** The most the gaps of two labels may differ by: every label is at the one distance. */
export const MAX_GAP_DIFFERENCE = 1.5

/** Whether the letters are beside the line, touching neither it nor the line's other side, and close to it. */
export const isHugging = (entry: LabelGap | undefined): boolean =>
  entry !== undefined && entry.gap > 0 && entry.gap <= MAX_LABEL_GAP

/** Whether two labels are at the same distance from their lines. */
export const isAtSameGap = (one: LabelGap | undefined, other: LabelGap | undefined): boolean =>
  one !== undefined && other !== undefined && Math.abs(one.gap - other.gap) < MAX_GAP_DIFFERENCE

/** The most the middle of the lowercase letters of a label on its line may be from it. */
export const MAX_CENTRE_OFFSET = 1

/** Whether the line passes through the middle of the lowercase letters: the one label of a line is set so. */
export const isOnLine = (entry: LabelGap | undefined): boolean =>
  entry !== undefined && entry.bandCentre < MAX_CENTRE_OFFSET

/**
 * Where the line and the letters of one label are in a picture of the screen, across the line, in
 * pixels from the curve at the middle of the label. Positive is to the right looking along the samples.
 */
export interface LabelPicture {
  /** The middle of the rows that have ink of the letters. */
  readonly ink: number
  /** The line where it leaves the halo before the letters, and where it comes out after them. */
  readonly before: number
  readonly after: number
  /** How many pixels of the line were found on each side. */
  readonly found: { readonly before: number; readonly after: number }
}

/**
 * Takes a picture of the label at the scale of the page and reads where the line enters and leaves
 * the letters, and where the ink of the letters is, in the same measure. The picture is of what is
 * painted, so it holds whatever the engine did with the baseline.
 */
export const readLabelPicture = async (
  page: Page,
  words: string,
  line: SampledLine,
): Promise<LabelPicture> => {
  const place = await page.evaluate(
    ({ words, points }) => {
      const svg = [...document.querySelectorAll<SVGSVGElement>('svg.plex')].find(
        (one) => one.getBoundingClientRect().width > 0,
      )!
      const letters = [...svg.querySelectorAll<SVGTextElement>('.plex__edge-label--letters')].find(
        (one) => one.textContent?.trim() === words,
      )!
      const box = letters.getBoundingClientRect()
      const middle = { x: box.x + box.width / 2, y: box.y + box.height / 2 }
      let nearest = 0
      points.forEach((point, step) => {
        const best = points[nearest]!
        if (
          Math.hypot(point.x - middle.x, point.y - middle.y) <
          Math.hypot(best.x - middle.x, best.y - middle.y)
        ) {
          nearest = step
        }
      })
      const back = points[Math.max(nearest - 1, 0)]!
      const on = points[Math.min(nearest + 1, points.length - 1)]!
      const size = Math.hypot(on.x - back.x, on.y - back.y) || 1
      const screen = letters.getScreenCTM()!
      const scale = Math.hypot(screen.a, screen.b)
      return {
        box: { x: box.x, y: box.y, width: box.width, height: box.height },
        origin: points[nearest]!,
        along: { x: (on.x - back.x) / size, y: (on.y - back.y) / size },
        width: letters.getComputedTextLength() * scale,
        halo: (Number.parseFloat(getComputedStyle(letters).strokeWidth) * scale) / 2,
      }
    },
    { words, points: line.points },
  )

  const pad = 24
  const clip = {
    x: Math.max(Math.floor(place.box.x - pad), 0),
    y: Math.max(Math.floor(place.box.y - pad), 0),
    width: Math.ceil(place.box.width + 2 * pad),
    height: Math.ceil(place.box.height + 2 * pad),
  }
  const picture = await page.screenshot({ clip, type: 'png' })
  const marks = await readMarks(page, picture.toString('base64'), clip, place)
  return getLabelPicture(marks, place)
}

/** The pixels of a picture that are not its ground, as screen offsets from a point, in plex terms along and across. */
interface Mark {
  readonly u: number
  readonly v: number
  readonly strength: number
}

interface Place {
  readonly origin: ScreenPoint
  readonly along: ScreenPoint
  readonly width: number
  readonly halo: number
}

/** Decodes the picture in the page and returns each pixel that differs from the ground, as along and across the line. */
const readMarks = (
  page: Page,
  data: string,
  clip: { x: number; y: number; width: number },
  place: Place,
): Promise<readonly Mark[]> =>
  page.evaluate(
    async ({ data, clip, place }) => {
      const bytes = Uint8Array.from(atob(data), (letter) => letter.charCodeAt(0))
      const bitmap = await createImageBitmap(new Blob([bytes], { type: 'image/png' }))
      const canvas = document.createElement('canvas')
      canvas.width = bitmap.width
      canvas.height = bitmap.height
      const paint = canvas.getContext('2d', { willReadFrequently: true })!
      paint.drawImage(bitmap, 0, 0)
      const { data: pixels } = paint.getImageData(0, 0, bitmap.width, bitmap.height)
      const ratio = bitmap.width / clip.width

      // The colour the picture is mostly made of is the ground.
      const counts = new Map<string, number>()
      for (let offset = 0; offset < pixels.length; offset += 4) {
        const key = `${pixels[offset]},${pixels[offset + 1]},${pixels[offset + 2]}`
        counts.set(key, (counts.get(key) ?? 0) + 1)
      }
      const ground = [...counts.entries()]
        .reduce((best, one) => (one[1] > best[1] ? one : best))[0]
        .split(',')
        .map(Number)

      const normal = { x: -place.along.y, y: place.along.x }
      const marks: { u: number; v: number; strength: number }[] = []
      for (let offset = 0; offset < pixels.length; offset += 4) {
        const strength =
          Math.abs(pixels[offset]! - ground[0]!) +
          Math.abs(pixels[offset + 1]! - ground[1]!) +
          Math.abs(pixels[offset + 2]! - ground[2]!)
        if (strength === 0) continue
        const pixel = offset / 4
        const x = clip.x + ((pixel % bitmap.width) + 0.5) / ratio - place.origin.x
        const y = clip.y + (Math.floor(pixel / bitmap.width) + 0.5) / ratio - place.origin.y
        marks.push({
          u: x * place.along.x + y * place.along.y,
          v: x * normal.x + y * normal.y,
          strength,
        })
      }
      return marks
    },
    { data, clip, place },
  )

/** The strongest mark of a list. */
const getStrongest = (list: readonly Mark[]): number =>
  list.reduce((most, mark) => Math.max(most, mark.strength), 0)

/** The marks at least half as strong as the strongest of them. */
const getStrong = (list: readonly Mark[]): Mark[] =>
  list.filter((mark) => mark.strength >= getStrongest(list) / 2)

/** The line where it is seen between two distances along it, weighted by how strongly it shows. */
const getSeen = (marks: readonly Mark[], from: number, to: number) => {
  const strong = getStrong(marks.filter((mark) => mark.u >= from && mark.u <= to))
  const total = strong.reduce((sum, mark) => sum + mark.strength, 0)
  return {
    at: strong.reduce((sum, mark) => sum + mark.v * mark.strength, 0) / (total || 1),
    count: strong.length,
  }
}

const REACH = 12

/** Where the ink and the line are in the marks of a picture, across the line. */
const getLabelPicture = (all: readonly Mark[], place: Place): LabelPicture => {
  const half = place.width / 2
  const close = all.filter((mark) => Math.abs(mark.v) <= REACH)
  const inked = getStrong(close.filter((mark) => Math.abs(mark.u) <= half))
  const vs = inked.map((mark) => mark.v)
  const start = half + place.halo + 2
  const before = getSeen(close, -start - 10, -start)
  const after = getSeen(close, start, start + 10)
  return {
    ink: (Math.min(...vs) + Math.max(...vs)) / 2,
    before: before.at,
    after: after.at,
    found: { before: before.count, after: after.count },
  }
}

/** The most the middle of the ink may be from the middle of the line that enters and leaves the letters. */
export const MAX_PICTURE_OFFSET = 1.5

/** Whether the line enters the letters and leaves them at the middle of their ink. */
export const isThroughTheMiddle = (picture: LabelPicture): boolean =>
  picture.found.before > 0 &&
  picture.found.after > 0 &&
  Math.abs(picture.ink - (picture.before + picture.after) / 2) <= MAX_PICTURE_OFFSET
