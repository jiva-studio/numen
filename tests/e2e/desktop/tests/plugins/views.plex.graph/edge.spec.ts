import {
  test,
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'

/** The note the vault opens with, and the child the fixture joins it to. */
const PATHS = { Thermodynamics: 'Thermodynamics.md', Entropy: 'notes/Entropy.md' } as const
type Name = keyof typeof PATHS

/** One record of a connection as the core answers it: fields at their zero value are left out. */
interface WireSide {
  readonly label?: string
  readonly marker?: 'MARKER_ARROW'
}

interface WireConnection {
  readonly first: string
  readonly second: string
  readonly firstSide?: WireSide
  readonly secondSide?: WireSide
}

const callNotes = async (request: APIRequestContext, method: string, data: object) => {
  const answer = await request.post(`/numen.v1.NoteService/${method}`, {
    data,
    headers: { 'Content-Type': 'application/json' },
  })
  if (!answer.ok()) throw new Error(`${method} answered ${answer.status()}`)
  return (await answer.json()) as Record<string, unknown>
}

/** Makes the records between Thermodynamics and Entropy exactly the ones given. */
const saveConnection = async (
  request: APIRequestContext,
  sides: { readonly Thermodynamics?: WireSide; readonly Entropy?: WireSide },
) => {
  const answer = await callNotes(request, 'SaveConnection', {
    connection: {
      first: PATHS.Thermodynamics,
      second: PATHS.Entropy,
      role: 'ROLE_CHILD',
      ...(sides.Thermodynamics ? { firstSide: sides.Thermodynamics } : {}),
      ...(sides.Entropy ? { secondSide: sides.Entropy } : {}),
    },
  })
  if (answer['error'] || answer['isInvalid'] || answer['isCycle']) {
    throw new Error(`SaveConnection refused: ${JSON.stringify(answer)}`)
  }
}

/** The record each note writes about the other, by the name of the note that writes it. */
const getRecords = async (request: APIRequestContext): Promise<Record<Name, WireSide | null>> => {
  const answer = await callNotes(request, 'GetConnection', {
    first: PATHS.Thermodynamics,
    second: PATHS.Entropy,
  })
  const connection = answer['connection'] as WireConnection
  // The role of each record is the core's to derive, so only what the inspector edits is compared.
  const getEdited = (side: WireSide | undefined): WireSide | null =>
    side
      ? {
          ...(side.label ? { label: side.label } : {}),
          ...(side.marker ? { marker: side.marker } : {}),
        }
      : null
  return {
    Thermodynamics: getEdited(connection.firstSide),
    Entropy: getEdited(connection.secondSide),
  }
}

const ARROW = 'MARKER_ARROW'

/** Opens the inspector on the line between Thermodynamics and Entropy, and the order it shows them in. */
const openLinkInspector = async (page: Page) => {
  await openWorkspace(page)
  await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

  const edgeHit = getGraph(page).locator('.plex__edge-hit').first()
  await expect(edgeHit).toBeAttached()
  await edgeHit.dispatchEvent('click', { clientX: 300, clientY: 300 })

  const popover = page.locator('.plex-link-popover')
  await expect(popover).toBeVisible()
  const nodeNames = (await popover.locator('.plex-link-popover__node-name').allTextContents()).map(
    (name) => name.trim(),
  )
  expect([...nodeNames].sort()).toStrictEqual(['Entropy', 'Thermodynamics'])
  const [first, second] = nodeNames as [Name, Name]
  return { popover, first, second }
}

const getDirections = (popover: Locator) =>
  popover
    .locator('.plex-link-row__dir-btn')
    .evaluateAll((buttons) => buttons.map((button) => button.getAttribute('data-direction')))

const clickDone = async (page: Page, popover: Locator) => {
  await popover.locator('.plex-link-popover__btn--done').click()
  await expect(popover).toBeHidden()
  await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()
}

interface Point {
  readonly x: number
  readonly y: number
}

interface Box {
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/** The line between two drawn notes: its end at each note, its middle, its labels and its heads, on the screen. */
interface EdgeGeometry {
  readonly ends: Readonly<Record<string, Point>>
  readonly middle: Point
  readonly labels: Readonly<Record<string, Point>>
  /** The box each label is drawn in, on the screen. */
  readonly boxes: Readonly<Record<string, Box>>
  /**
   * How far the middle of each label is from the line, across it, in pixels: positive on the
   * right of the line looking from the first named note to the second, negative on the left.
   */
  readonly across: Readonly<Record<string, number>>
  /**
   * How much each label is read in the direction from the first named note to the second: 1 along
   * the line that way, -1 along it the other way, 0 across it.
   */
  readonly reading: Readonly<Record<string, number>>
  readonly arrows: readonly Point[]
}

/**
 * Reads the line between two notes off the drawing. The line is the one whose ends lie nearest
 * the two titles; each label and each head is measured where the browser drew it.
 */
const readEdge = (page: Page, names: readonly [Name, Name]): Promise<EdgeGeometry> =>
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
      const points = Array.from({ length: Math.ceil(length / 2) + 1 }, (_, step) =>
        getPoint(Math.min(2 * step, length)),
      )
      return { start: getPoint(0), end: getPoint(length), middle: getPoint(length / 2), points }
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
    const boxes: Record<string, Box> = {}
    const offsets: Record<string, number> = {}
    const directions: Record<string, number> = {}
    const way = isStartAtFirst ? 1 : -1
    for (const letters of svg.querySelectorAll('.plex__edge-label--letters')) {
      const box = letters.getBoundingClientRect()
      const words = letters.textContent?.trim() ?? ''
      labels[words] = centre(box)
      boxes[words] = { x: box.x, y: box.y, width: box.width, height: box.height }

      // Measure against the line where it is nearest the label, taken from the first note to the second.
      const middle = centre(box)
      const nearest = line.points.reduce(
        (best, point, step) =>
          distance(point, middle) < distance(line.points[best]!, middle) ? step : best,
        0,
      )
      const here = line.points[nearest]!
      const back = line.points[Math.max(nearest - 1, 0)]!
      const on = line.points[Math.min(nearest + 1, line.points.length - 1)]!
      const run = { x: (on.x - back.x) * way, y: (on.y - back.y) * way }
      const size = Math.hypot(run.x, run.y) || 1
      offsets[words] = ((middle.x - here.x) * -run.y + (middle.y - here.y) * run.x) / size
      const turn = ((letters as SVGTextContentElement).getRotationOfChar(0) * Math.PI) / 180
      directions[words] = (Math.cos(turn) * run.x + Math.sin(turn) * run.y) / size
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
      boxes,
      across: offsets,
      reading: directions,
      arrows,
    }
  }, names)

const getDistance = (one: Point, other: Point) => Math.hypot(one.x - other.x, one.y - other.y)

/**
 * How far along the line a point is, as a fraction of the way from the end at `from` to the end
 * at `to`, taking the point's projection on the straight run between the two ends.
 */
const getShare = (edge: EdgeGeometry, point: Point, from: Name, to: Name) => {
  const start = edge.ends[from]!
  const run = { x: edge.ends[to]!.x - start.x, y: edge.ends[to]!.y - start.y }
  return ((point.x - start.x) * run.x + (point.y - start.y) * run.y) / (run.x ** 2 + run.y ** 2)
}

const isApart = (one: Box, other: Box) =>
  one.x + one.width <= other.x ||
  other.x + other.width <= one.x ||
  one.y + one.height <= other.y ||
  other.y + other.height <= one.y

/** The label is in the middle of the line, within a tenth of the line either way. */
const isCentred = (edge: EdgeGeometry, label: Point) =>
  Math.abs(getShare(edge, label, 'Thermodynamics', 'Entropy') - 0.5) < 0.1

/** Whether a point is nearer the end of the line at `nearName` than the end at `farName`. */
const isNearer = (edge: EdgeGeometry, point: Point, nearName: Name, farName: Name) =>
  getDistance(point, edge.ends[nearName]!) < getDistance(point, edge.ends[farName]!)

const NAMES = ['Thermodynamics', 'Entropy'] as const

/**
 * The two labels are on opposite sides of the line and clear of it, the record of the parent on
 * the left looking from the parent and read from the parent to the child, the record of the child
 * on the right and read from the child to the parent.
 */
const isOnItsSide = (edge: EdgeGeometry) =>
  edge.across['flows into']! < -5 &&
  edge.across['measures']! > 5 &&
  edge.reading['flows into']! > 0.5 &&
  edge.reading['measures']! < -0.5

test.describe('Plex Edge and Link Inspector', () => {
  test.beforeEach(async ({ request }) => {
    await saveConnection(request, { Thermodynamics: { label: 'warms' } })
  })

  test.afterEach(async ({ request }) => {
    await saveConnection(request, { Thermodynamics: { marker: ARROW } })
  })

  test('shows one row with a direction button, a description and a trash button', async ({
    page,
  }) => {
    const { popover } = await openLinkInspector(page)

    await expect(popover.locator('.plex-link-popover__title')).toBeVisible()
    await expect(popover.locator('.plex-link-row')).toHaveCount(1)
    await expect(popover.locator('.plex-link-row__dir-btn')).toHaveCount(1)
    await expect(popover.locator('.plex-link-row__input')).toHaveValue('warms')
    await expect(popover.locator('.plex-link-row__delete-btn')).toHaveCount(1)
    await expect(popover.locator('select')).toHaveCount(0)
    await expect(popover.locator('input[type="checkbox"]')).toHaveCount(0)
    await expect(popover.locator('.plex-link-popover__btn--add')).toContainText(
      'Add reverse direction',
    )
    expect(await getDirections(popover)).toStrictEqual(['none'])
  })

  test('cycles the direction through none, forward and backward and saves each one', async ({
    page,
    request,
  }) => {
    test.setTimeout(120_000)

    // Forward: the first note writes the record and points at the second.
    let { popover, first, second } = await openLinkInspector(page)
    await popover.locator('.plex-link-row__dir-btn').click()
    expect(await getDirections(popover)).toStrictEqual(['forward'])
    await clickDone(page, popover)

    expect(await getRecords(request)).toEqual({
      [first]: { label: 'warms', marker: ARROW },
      [second]: null,
    })
    await expect
      .poll(async () => {
        const edge = await readEdge(page, NAMES)
        return (
          edge.arrows.length === 1 &&
          isNearer(edge, edge.arrows[0]!, second, first) &&
          isCentred(edge, edge.labels['warms']!)
        )
      })
      .toBe(true)

    // Backward: the record moves to the second note, with its label, and points at the first.
    ;({ popover, first, second } = await openLinkInspector(page))
    expect(await getDirections(popover)).toStrictEqual(['forward'])
    await popover.locator('.plex-link-row__dir-btn').click()
    expect(await getDirections(popover)).toStrictEqual(['backward'])
    await clickDone(page, popover)

    expect(await getRecords(request)).toEqual({
      [first]: null,
      [second]: { label: 'warms', marker: ARROW },
    })
    await expect
      .poll(async () => {
        const edge = await readEdge(page, NAMES)
        return (
          edge.arrows.length === 1 &&
          isNearer(edge, edge.arrows[0]!, first, second) &&
          isCentred(edge, edge.labels['warms']!)
        )
      })
      .toBe(true)

    // None: the record goes back to the first note without a marker.
    ;({ popover, first, second } = await openLinkInspector(page))
    await popover.locator('.plex-link-row__dir-btn').click()
    expect(await getDirections(popover)).toStrictEqual(['none'])
    await clickDone(page, popover)

    expect(await getRecords(request)).toEqual({ [first]: { label: 'warms' }, [second]: null })
    await expect.poll(async () => (await readEdge(page, NAMES)).arrows.length).toBe(0)
  })

  test('sets the label of a record without a direction in the middle of its line', async ({
    page,
  }) => {
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    await expect
      .poll(async () => {
        const edge = await readEdge(page, NAMES)
        const label = edge.labels['warms']
        if (!label) return false
        const length = getDistance(edge.ends['Thermodynamics']!, edge.ends['Entropy']!)
        const fromEnds = NAMES.map((name) => getDistance(label, edge.ends[name]!))
        return (
          getDistance(label, edge.middle) < length / 4 &&
          Math.abs(fromEnds[0]! - fromEnds[1]!) < length / 4
        )
      })
      .toBe(true)
  })

  test('adds the reverse direction with its button and saves two descriptions', async ({
    page,
    request,
  }) => {
    const { popover, first, second } = await openLinkInspector(page)

    await popover.locator('.plex-link-popover__btn--add').click()
    expect(await getDirections(popover)).toStrictEqual(['forward', 'backward'])
    await expect(popover.locator('.plex-link-row__dir-btn').first()).toBeDisabled()
    await expect(popover.locator('.plex-link-popover__btn--add')).toHaveCount(0)

    const inputs = popover.locator('.plex-link-row__input')
    await inputs.first().fill('flows into')
    await inputs.last().fill('measures')
    await clickDone(page, popover)

    expect(await getRecords(request)).toEqual({
      [first]: { label: 'flows into', marker: ARROW },
      [second]: { label: 'measures', marker: ARROW },
    })
    await expect
      .poll(async () => {
        const edge = await readEdge(page, NAMES)
        const heads = edge.arrows
        return (
          heads.length === 2 &&
          heads.some((head) => isNearer(edge, head, first, second)) &&
          heads.some((head) => isNearer(edge, head, second, first)) &&
          isNearer(edge, edge.labels['flows into']!, 'Thermodynamics', 'Entropy') &&
          isNearer(edge, edge.labels['measures']!, 'Entropy', 'Thermodynamics')
        )
      })
      .toBe(true)
  })

  test('sets the two labels of a line apart, one at each end, on opposite sides, and clear of each other', async ({
    page,
    request,
  }) => {
    await saveConnection(request, {
      Thermodynamics: { label: 'flows into', marker: ARROW },
      Entropy: { label: 'measures', marker: ARROW },
    })
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    // The record of the parent is by the parent, in its first third; the record of the child by the
    // child, in its last, on the other side of the line.
    await expect
      .poll(async () => {
        const edge = await readEdge(page, NAMES)
        const parentLabel = edge.labels['flows into']
        const childLabel = edge.labels['measures']
        if (!parentLabel || !childLabel) return false
        return (
          getShare(edge, parentLabel, 'Thermodynamics', 'Entropy') < 1 / 3 &&
          getShare(edge, childLabel, 'Thermodynamics', 'Entropy') > 2 / 3 &&
          isApart(edge.boxes['flows into']!, edge.boxes['measures']!) &&
          isOnItsSide(edge)
        )
      })
      .toBe(true)
  })

  test('sets the two labels at the two ends whatever the line has for heads', async ({
    page,
    request,
  }) => {
    await saveConnection(request, {
      Thermodynamics: { label: 'flows into' },
      Entropy: { label: 'measures' },
    })
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    await expect
      .poll(async () => {
        const edge = await readEdge(page, NAMES)
        const parentLabel = edge.labels['flows into']
        const childLabel = edge.labels['measures']
        if (!parentLabel || !childLabel || edge.arrows.length !== 0) return false
        return (
          getShare(edge, parentLabel, 'Thermodynamics', 'Entropy') < 1 / 3 &&
          getShare(edge, childLabel, 'Thermodynamics', 'Entropy') > 2 / 3 &&
          isOnItsSide(edge)
        )
      })
      .toBe(true)
  })

  test('centres the one label of a line that has a head at one end', async ({ page, request }) => {
    await saveConnection(request, { Thermodynamics: { label: 'flows into', marker: ARROW } })
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    await expect
      .poll(async () => {
        const edge = await readEdge(page, NAMES)
        const label = edge.labels['flows into']
        return (
          !!label &&
          edge.arrows.length === 1 &&
          isCentred(edge, label) &&
          Math.abs(edge.across['flows into']!) < 5
        )
      })
      .toBe(true)
  })

  test('adds the reverse direction with Ctrl+Enter', async ({ page, request }) => {
    const { popover } = await openLinkInspector(page)
    const inputs = popover.locator('.plex-link-row__input')

    await inputs.first().press('Control+Enter')
    await expect(popover.locator('.plex-link-row')).toHaveCount(2)
    expect(await getDirections(popover)).toStrictEqual(['forward', 'backward'])

    // The added record is the one with an empty description.
    const values = await inputs.evaluateAll((fields) =>
      fields.map((field) => (field as HTMLInputElement).value),
    )
    expect([...values].sort()).toStrictEqual(['', 'warms'])
    await inputs.evaluateAll((fields) =>
      (fields as HTMLInputElement[]).find((field) => field.value === '')?.focus(),
    )
    await page.keyboard.insertText('measures')
    await page.keyboard.press('Enter')
    await expect(popover).toBeHidden()

    expect(await getRecords(request)).toEqual({
      Thermodynamics: { label: 'warms', marker: ARROW },
      Entropy: { label: 'measures', marker: ARROW },
    })
  })

  test('removes one direction with its trash button', async ({ page, request }) => {
    const both = {
      Thermodynamics: { label: 'flows into', marker: ARROW },
      Entropy: { label: 'measures', marker: ARROW },
    } as const
    await saveConnection(request, both)
    const { popover, first, second } = await openLinkInspector(page)
    expect(await getDirections(popover)).toStrictEqual(['forward', 'backward'])

    await popover.locator('.plex-link-row__delete-btn').last().click()
    await expect(popover.locator('.plex-link-row')).toHaveCount(1)
    expect(await getDirections(popover)).toStrictEqual(['forward'])
    await clickDone(page, popover)

    expect(await getRecords(request)).toEqual({ [first]: both[first], [second]: null })
    await expect.poll(async () => (await readEdge(page, NAMES)).arrows.length).toBe(1)
  })

  test('dismisses link inspector via close button without breaking plex graph', async ({
    page,
    request,
  }) => {
    const { popover } = await openLinkInspector(page)

    await popover.locator('.plex-link-row__dir-btn').click()
    const closeBtn = popover.locator('.plex-link-popover__close-btn')
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()

    await expect(popover).toBeHidden()
    expect(await getRecords(request)).toEqual({
      Thermodynamics: { label: 'warms' },
      Entropy: null,
    })

    const thermoNode = page.locator('.plex__title', { hasText: 'Thermodynamics' })
    await expect(thermoNode).toBeVisible()
  })
})
