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

/** The line between two drawn notes: its end at each note, its middle, its labels and its heads, on the screen. */
interface EdgeGeometry {
  readonly ends: Readonly<Record<string, Point>>
  readonly middle: Point
  readonly labels: Readonly<Record<string, Point>>
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

/** Whether a point is nearer the end of the line at `nearName` than the end at `farName`. */
const isNearer = (edge: EdgeGeometry, point: Point, nearName: Name, farName: Name) =>
  getDistance(point, edge.ends[nearName]!) < getDistance(point, edge.ends[farName]!)

const NAMES = ['Thermodynamics', 'Entropy'] as const

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
          isNearer(edge, edge.labels['warms']!, second, first)
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
          isNearer(edge, edge.labels['warms']!, first, second)
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
          isNearer(edge, edge.labels['flows into']!, second, first) &&
          isNearer(edge, edge.labels['measures']!, first, second)
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
    const added = values.indexOf('') === 0 ? inputs.first() : inputs.last()
    await added.fill('measures')
    await added.press('Enter')
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
