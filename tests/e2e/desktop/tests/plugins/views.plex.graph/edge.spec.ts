import {
  test,
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import {
  isAtSameGap,
  isHugging,
  isOnLine,
  isThroughTheMiddle,
  readLabelPicture,
} from '../../support/plex-labels.js'
import { getDistance, readPlexLine, type Box, type LineGeometry, type Point } from '../../support/plex-lines.js'

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

/**
 * Presses the direction button of a row with the mouse at the middle of the button, after checking
 * that the button is the topmost element there: nothing covers it, and it is not disabled.
 */
const getDirectionButton = async (popover: Locator, row: number) =>
  (await popover.locator('.plex-link-row__dir-btn').all())[row]!

const pressDirection = async (page: Page, popover: Locator, row: number) => {
  const button = await getDirectionButton(popover, row)
  await expect(button).toBeEnabled()
  const centre = await button.evaluate((element) => {
    const box = element.getBoundingClientRect()
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    const top = document.elementFromPoint(x, y)
    return { x, y, isOnTop: top !== null && element.contains(top) }
  })
  expect(centre.isOnTop).toBe(true)
  await page.mouse.click(centre.x, centre.y)
}

const getButtonName = async (popover: Locator, row: number) =>
  (await getDirectionButton(popover, row)).getAttribute('aria-label')

const clickDone = async (page: Page, popover: Locator) => {
  await popover.locator('.plex-link-popover__btn--done').click()
  await expect(popover).toBeHidden()
  await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()
}

/**
 * How far along the line a point is, as a fraction of the way from the end at `from` to the end
 * at `to`, taking the point's projection on the straight run between the two ends.
 */
const getShare = (edge: LineGeometry, point: Point, from: Name, to: Name) => {
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
const isCentred = (edge: LineGeometry, label: Point) =>
  Math.abs(getShare(edge, label, 'Thermodynamics', 'Entropy') - 0.5) < 0.1

/** Whether a point is nearer the end of the line at `nearName` than the end at `farName`. */
const isNearer = (edge: LineGeometry, point: Point, nearName: Name, farName: Name) =>
  getDistance(point, edge.ends[nearName]!) < getDistance(point, edge.ends[farName]!)

const NAMES = ['Thermodynamics', 'Entropy'] as const

/**
 * The two labels are on opposite sides of the line and clear of it, the record of the parent on
 * the left looking from the parent and read from the parent to the child, the record of the child
 * on the right and read from the child to the parent.
 */
const isOnItsSide = (edge: LineGeometry) =>
  edge.gaps['flows into']!.side < 0 &&
  edge.gaps['measures']!.side > 0 &&
  isHugging(edge.gaps['flows into']) &&
  isHugging(edge.gaps['measures']) &&
  isAtSameGap(edge.gaps['flows into'], edge.gaps['measures']) &&
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
        const edge = await readPlexLine(page, NAMES)
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
        const edge = await readPlexLine(page, NAMES)
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
    await expect.poll(async () => (await readPlexLine(page, NAMES)).arrows.length).toBe(0)
  })

  test('sets the label of a record without a direction in the middle of its line', async ({
    page,
  }) => {
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    await expect
      .poll(async () => {
        const edge = await readPlexLine(page, NAMES)
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
    await expect(popover.locator('.plex-link-row__dir-btn').first()).toBeEnabled()
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
        const edge = await readPlexLine(page, NAMES)
        const heads = edge.arrows
        return (
          heads.length === 2 &&
          heads.some((head) => isNearer(edge, head, first, second)) &&
          heads.some((head) => isNearer(edge, head, second, first)) &&
          isNearer(edge, edge.labels['flows into']!, 'Entropy', 'Thermodynamics') &&
          isNearer(edge, edge.labels['measures']!, 'Thermodynamics', 'Entropy')
        )
      })
      .toBe(true)
  })

  test('sets each of two labels near the end its own head is at, reads it toward that end, on opposite sides', async ({
    page,
    request,
  }) => {
    await saveConnection(request, {
      Thermodynamics: { label: 'flows into', marker: ARROW },
      Entropy: { label: 'measures', marker: ARROW },
    })
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    // The record of the parent points at the child: it is in the last third, by the child, and reads
    // toward it. The record of the child points at the parent: it is in the first third, by the
    // parent, reads toward it, and is on the other side of the line.
    await expect
      .poll(async () => {
        const edge = await readPlexLine(page, NAMES)
        const parentLabel = edge.labels['flows into']
        const childLabel = edge.labels['measures']
        if (!parentLabel || !childLabel) return false
        return (
          getShare(edge, parentLabel, 'Thermodynamics', 'Entropy') > 2 / 3 &&
          getShare(edge, childLabel, 'Thermodynamics', 'Entropy') < 1 / 3 &&
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
        const edge = await readPlexLine(page, NAMES)
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

  test('centres the one label of a line that has a head at one end, along the line and on it', async ({
    page,
    request,
  }) => {
    await saveConnection(request, { Thermodynamics: { label: 'flows into', marker: ARROW } })
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    await expect
      .poll(async () => {
        const edge = await readPlexLine(page, NAMES)
        const label = edge.labels['flows into']
        return (
          !!label &&
          edge.arrows.length === 1 &&
          isCentred(edge, label) &&
          isOnLine(edge.gaps['flows into'])
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
    await expect.poll(async () => (await readPlexLine(page, NAMES)).arrows.length).toBe(1)
  })

  test('turns the arrows of two records already on disk with real clicks, one record each', async ({
    page,
    request,
  }) => {
    await saveConnection(request, {
      Thermodynamics: { label: 'Lopata' },
      Entropy: { label: 'ererer' },
    })
    const { popover, first, second } = await openLinkInspector(page)
    expect(await getDirections(popover)).toStrictEqual(['none', 'none'])
    expect(await getButtonName(popover, 0)).toBe(`No arrow. Click for: Arrow to ${second}`)
    expect(await getButtonName(popover, 1)).toBe(`No arrow. Click for: Arrow to ${first}`)

    await pressDirection(page, popover, 1)
    expect(await getDirections(popover)).toStrictEqual(['none', 'backward'])
    expect(await getButtonName(popover, 1)).toBe(`Arrow to ${first}. Click for: No arrow`)
    await pressDirection(page, popover, 0)
    expect(await getDirections(popover)).toStrictEqual(['forward', 'backward'])
    expect(await getButtonName(popover, 0)).toBe(`Arrow to ${second}. Click for: No arrow`)
    await pressDirection(page, popover, 1)
    expect(await getDirections(popover)).toStrictEqual(['forward', 'none'])
    await clickDone(page, popover)

    const records = await getRecords(request)
    expect(records[first]).toEqual({ label: records[first]!.label, marker: ARROW })
    expect(records[second]).toEqual({ label: records[second]!.label })
    await expect.poll(async () => (await readPlexLine(page, NAMES)).arrows.length).toBe(1)
  })

  test('takes the arrows off two directed records on disk with real clicks', async ({
    page,
    request,
  }) => {
    await saveConnection(request, {
      Thermodynamics: { label: 'flows into', marker: ARROW },
      Entropy: { label: 'measures', marker: ARROW },
    })
    const { popover, first, second } = await openLinkInspector(page)
    expect(await getDirections(popover)).toStrictEqual(['forward', 'backward'])

    await pressDirection(page, popover, 0)
    expect(await getDirections(popover)).toStrictEqual(['none', 'backward'])
    await pressDirection(page, popover, 1)
    expect(await getDirections(popover)).toStrictEqual(['none', 'none'])
    await clickDone(page, popover)

    const records = await getRecords(request)
    expect(Object.values(records).map((side) => side?.marker)).toStrictEqual([undefined, undefined])
    expect(records[first]?.label).toBeTruthy()
    expect(records[second]?.label).toBeTruthy()
    await expect.poll(async () => (await readPlexLine(page, NAMES)).arrows.length).toBe(0)
  })

  test('cycles a lone record with real clicks and names each arrow and the next', async ({
    page,
    request,
  }) => {
    const { popover, first, second } = await openLinkInspector(page)
    const names: (string | null)[] = [await getButtonName(popover, 0)]
    for (let step = 0; step < 3; step += 1) {
      await pressDirection(page, popover, 0)
      names.push(await getButtonName(popover, 0))
    }
    expect(names).toStrictEqual([
      `No arrow. Click for: Arrow to ${second}`,
      `Arrow to ${second}. Click for: Arrow to ${first}`,
      `Arrow to ${first}. Click for: No arrow`,
      `No arrow. Click for: Arrow to ${second}`,
    ])
    await pressDirection(page, popover, 0)
    await clickDone(page, popover)
    expect(await getRecords(request)).toEqual({
      [first]: { label: 'warms', marker: ARROW },
      [second]: null,
    })
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

test.describe('The one label of a line, at four pixels to a pixel', () => {
  test.use({ deviceScaleFactor: 4 })

  test.afterEach(async ({ request }) => {
    await saveConnection(request, { Thermodynamics: { marker: ARROW } })
  })

  test('has the middle of its letters on the line, which enters and leaves the text at that middle', async ({
    page,
    request,
  }) => {
    await saveConnection(request, { Thermodynamics: { label: 'eee' } })
    await openWorkspace(page)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()

    await expect
      .poll(async () => {
        const edge = await readPlexLine(page, NAMES)
        return edge.gaps['eee'] !== undefined && isCentred(edge, edge.labels['eee']!)
      })
      .toBe(true)

    // Every figure is checked and reported, so a failure shows where the letters stand.
    const edge = await readPlexLine(page, NAMES)
    const picture = await readLabelPicture(page, 'eee', edge.line)
    const middle = (picture.before + picture.after) / 2
    const report = JSON.stringify({ gap: edge.gaps['eee'], picture })

    // The ink of the letters as read off pixels is within a pixel of the line, in either engine.
    expect.soft(edge.gaps['eee']!.centre, report).toBeLessThan(1)
    expect.soft(edge.gaps['eee']!.bandCentre, report).toBeLessThan(1)

    // In the picture of the screen the line goes into the text and comes out of it at its middle.
    expect.soft(picture.found.before, report).toBeGreaterThan(0)
    expect.soft(picture.found.after, report).toBeGreaterThan(0)
    expect.soft(Math.abs(picture.ink - middle), report).toBeLessThanOrEqual(1.5)
    expect.soft(isThroughTheMiddle(picture), report).toBe(true)
  })
})
