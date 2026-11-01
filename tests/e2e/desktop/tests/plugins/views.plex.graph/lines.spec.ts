import { test, expect, type APIRequestContext, type Page } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import { createNoteWithBody, destroyNotes, getRow, showInFiles } from '../../support/layout.js'
import { callRPC } from '../../support/vault.js'

/** One record of a connection, as SaveConnection takes it. */
interface WireSide {
  readonly label?: string
  readonly marker?: 'MARKER_ARROW'
}

/** Saves the connection between two notes with the records given. */
const saveConnection = async (
  request: APIRequestContext,
  connection: {
    readonly first: string
    readonly second: string
    readonly role: 'ROLE_CHILD' | 'ROLE_JUMP'
    readonly firstSide?: WireSide
    readonly secondSide?: WireSide
  },
): Promise<void> => {
  const answer = await callRPC(request, 'NoteService/SaveConnection', { connection })
  if (answer['error'] || answer['isInvalid'] || answer['isCycle']) {
    throw new Error(`SaveConnection refused: ${JSON.stringify(answer)}`)
  }
}

/** Shows the note at the path in the plex, from the menu on its row in the files tab. */
const showInPlex = async (page: Page, path: string): Promise<void> => {
  await openWorkspace(page)
  await showInFiles(page, ['notes'])
  await getRow(page, path).click({ button: 'right' })
  await page.getByRole('menuitem', { name: 'Show in plex', exact: true }).click()
  await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()
}

interface Point {
  readonly x: number
  readonly y: number
}

/** The line between two drawn notes: its end at each note, its middle, and the labels and heads of the plex. */
interface LineGeometry {
  readonly ends: Readonly<Record<string, Point>>
  readonly middle: Point
  readonly labels: Readonly<Record<string, Point>>
  readonly arrows: readonly Point[]
}

/**
 * Reads the line between two notes off the drawing. The line is the one whose ends lie nearest
 * the two titles; each label and each head is measured where the browser drew it.
 */
const readLine = (page: Page, names: readonly [string, string]): Promise<LineGeometry> =>
  page.evaluate((names) => {
    const svg = [...document.querySelectorAll<SVGSVGElement>('svg.plex')].find(
      (one) => one.getBoundingClientRect().width > 0,
    )!
    const centre = (box: DOMRect) => ({ x: box.x + box.width / 2, y: box.y + box.height / 2 })
    const distance = (one: { x: number; y: number }, other: { x: number; y: number }) =>
      Math.hypot(one.x - other.x, one.y - other.y)
    const titles = names.map((name) => {
      const title = [...svg.querySelectorAll('.plex__title')].find(
        (one) => one.textContent?.trim() === name,
      )!
      return centre(title.getBoundingClientRect())
    })

    const lines = [...svg.querySelectorAll<SVGPathElement>('.plex__edge')].map((path) => {
      const screen = path.getScreenCTM()!
      const length = path.getTotalLength()
      const getPoint = (offset: number) => {
        const point = path.getPointAtLength(offset).matrixTransform(screen)
        return { x: point.x, y: point.y }
      }
      return { start: getPoint(0), end: getPoint(length), middle: getPoint(length / 2) }
    })
    const getFit = (line: (typeof lines)[number]) =>
      Math.min(
        distance(line.start, titles[0]!) + distance(line.end, titles[1]!),
        distance(line.end, titles[0]!) + distance(line.start, titles[1]!),
      )
    const line = lines.reduce((best, one) => (getFit(one) < getFit(best) ? one : best))
    const isStartAtFirst =
      distance(line.start, titles[0]!) + distance(line.end, titles[1]!) <=
      distance(line.end, titles[0]!) + distance(line.start, titles[1]!)

    const labels: Record<string, { x: number; y: number }> = {}
    for (const letters of svg.querySelectorAll('.plex__edge-label--letters')) {
      labels[letters.textContent?.trim() ?? ''] = centre(letters.getBoundingClientRect())
    }
    const arrows = [...svg.querySelectorAll('.plex__edge-arrow')].map((head) =>
      centre(head.getBoundingClientRect()),
    )
    return {
      ends: {
        [names[0]]: isStartAtFirst ? line.start : line.end,
        [names[1]]: isStartAtFirst ? line.end : line.start,
      },
      middle: line.middle,
      labels,
      arrows,
    }
  }, names)

const getDistance = (one: Point, other: Point) => Math.hypot(one.x - other.x, one.y - other.y)

/**
 * Whether the line between two notes has the only arrowhead of the plex at the end of `target`,
 * with the words `words` set nearer that end than the other.
 */
const isPointingAt = async (
  page: Page,
  names: readonly [string, string],
  target: string,
  words: string,
): Promise<boolean> => {
  const line = await readLine(page, names)
  const other = names.find((name) => name !== target)!
  const label = line.labels[words]
  return (
    line.arrows.length === 1 &&
    label !== undefined &&
    getDistance(line.arrows[0]!, line.ends[target]!) <
      getDistance(line.arrows[0]!, line.ends[other]!) &&
    getDistance(label, line.ends[target]!) < getDistance(label, line.ends[other]!)
  )
}

/** Whether the words of a record without an arrowhead are set in the middle of the line. */
const isSetInTheMiddle = async (
  page: Page,
  names: readonly [string, string],
  words: string,
): Promise<boolean> => {
  const line = await readLine(page, names)
  const label = line.labels[words]
  const length = getDistance(line.ends[names[0]]!, line.ends[names[1]]!)
  return label !== undefined && getDistance(label, line.middle) < length / 4
}

test.describe('Lines between notes other than the one in focus', () => {
  let notes: Record<string, string> = {}

  test.beforeEach(async ({ request }) => {
    notes = {}
    for (const title of ['Pharos', 'Ember', 'Flare', 'Headland', 'Lamp', 'Mirror']) {
      notes[title] = await createNoteWithBody(request, title, 'notes', `# ${title}\n`)
    }
  })

  test.afterEach(async ({ request }) => {
    await destroyNotes(request, Object.values(notes))
  })

  test('draws the arrowhead and the words of a record on a line between two children of the focus', async ({
    page,
    request,
  }) => {
    await saveConnection(request, {
      first: notes['Pharos']!,
      second: notes['Ember']!,
      role: 'ROLE_CHILD',
      firstSide: { label: 'kindles' },
    })
    await saveConnection(request, {
      first: notes['Pharos']!,
      second: notes['Flare']!,
      role: 'ROLE_CHILD',
      firstSide: { label: 'signals' },
    })
    await saveConnection(request, {
      first: notes['Ember']!,
      second: notes['Flare']!,
      role: 'ROLE_JUMP',
      firstSide: { label: 'feeds', marker: 'MARKER_ARROW' },
    })

    await showInPlex(page, notes['Pharos']!)
    await expect(getGraph(page).getByRole('button', { name: 'Flare, child' })).toBeAttached()

    await expect.poll(() => isPointingAt(page, ['Ember', 'Flare'], 'Flare', 'feeds')).toBe(true)
    await expect.poll(() => isSetInTheMiddle(page, ['Pharos', 'Ember'], 'kindles')).toBe(true)
  })

  test('draws the arrowhead and the words of a record on a line between two siblings of the focus', async ({
    page,
    request,
  }) => {
    for (const [child, words] of [
      ['Pharos', 'holds'],
      ['Lamp', 'raises'],
      ['Mirror', 'carries'],
    ] as const) {
      await saveConnection(request, {
        first: notes['Headland']!,
        second: notes[child]!,
        role: 'ROLE_CHILD',
        firstSide: { label: words },
      })
    }
    await saveConnection(request, {
      first: notes['Lamp']!,
      second: notes['Mirror']!,
      role: 'ROLE_JUMP',
      firstSide: { label: 'faces', marker: 'MARKER_ARROW' },
    })

    await showInPlex(page, notes['Pharos']!)
    await expect(getGraph(page).getByRole('button', { name: 'Mirror, sibling' })).toBeAttached()

    await expect.poll(() => isPointingAt(page, ['Lamp', 'Mirror'], 'Mirror', 'faces')).toBe(true)
    await expect.poll(() => isSetInTheMiddle(page, ['Headland', 'Lamp'], 'raises')).toBe(true)
  })
})
