import { test, expect, type APIRequestContext, type Page } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import { createNoteWithBody, destroyNotes, getRow, showInFiles } from '../../support/layout.js'
import { callRPC } from '../../support/vault.js'
import { isAtSameGap, isHugging, isOnLine } from '../../support/plex-labels.js'
import { getDistance, readPlexLine, type LineGeometry, type Point } from '../../support/plex-lines.js'

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

/**
 * Whether the line between two notes has the only arrowhead of the plex at the end of `target`,
 * with its one label `words` in the middle of the line, within a tenth of its length.
 */
const isPointingAt = async (
  page: Page,
  names: readonly [string, string],
  target: string,
  words: string,
): Promise<boolean> => {
  const line = await readPlexLine(page, names)
  const other = names.find((name) => name !== target)!
  const label = line.labels[words]
  return (
    line.arrows.length === 1 &&
    label !== undefined &&
    getDistance(line.arrows[0]!, line.ends[target]!) <
      getDistance(line.arrows[0]!, line.ends[other]!) &&
    getDistance(label, line.middle) < getDistance(line.ends[target]!, line.ends[other]!) / 10
  )
}

/**
 * Whether a label hugs the line on the side its reading puts it: read toward the end it is by
 * (`way` 1 toward the second note, -1 toward the first), on the left of its own reading.
 */
const isReadToward = (line: LineGeometry, words: string, way: 1 | -1): boolean =>
  isHugging(line.gaps[words]) && line.gaps[words]!.side === -way && line.reading[words]! * way > 0.5

/**
 * Whether the line between two notes carries the label `toFirst`, the record that points at
 * `names[0]`, in its first third, by that note and read toward it, and the label `toSecond`, the
 * record that points at `names[1]`, in its last third, by that note and read toward it, the two
 * boxes clear of each other, on opposite sides of the line, at the same distance from it.
 */
const isSetAtBothEnds = async (
  page: Page,
  names: readonly [string, string],
  words: { readonly toFirst: string; readonly toSecond: string },
): Promise<boolean> => {
  const line = await readPlexLine(page, names)
  const first = line.labels[words.toFirst]
  const last = line.labels[words.toSecond]
  if (!first || !last) return false
  const start = line.ends[names[0]]!
  const run = { x: line.ends[names[1]]!.x - start.x, y: line.ends[names[1]]!.y - start.y }
  const getShare = (point: Point) =>
    ((point.x - start.x) * run.x + (point.y - start.y) * run.y) / (run.x ** 2 + run.y ** 2)
  const one = line.boxes[words.toFirst]!
  const other = line.boxes[words.toSecond]!
  const isApart =
    one.x + one.width <= other.x ||
    other.x + other.width <= one.x ||
    one.y + one.height <= other.y ||
    other.y + other.height <= one.y
  return (
    getShare(first) < 1 / 3 &&
    getShare(last) > 2 / 3 &&
    isApart &&
    isReadToward(line, words.toFirst, -1) &&
    isReadToward(line, words.toSecond, 1) &&
    isAtSameGap(line.gaps[words.toFirst], line.gaps[words.toSecond])
  )
}

/** Whether the words of a record without an arrowhead are set in the middle of the line. */
const isSetInTheMiddle = async (
  page: Page,
  names: readonly [string, string],
  words: string,
): Promise<boolean> => {
  const line = await readPlexLine(page, names)
  const label = line.labels[words]
  const length = getDistance(line.ends[names[0]]!, line.ends[names[1]]!)
  return (
    label !== undefined &&
    getDistance(label, line.middle) < length / 4 &&
    isOnLine(line.gaps[words])
  )
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

  test('sets the two records of a line between two siblings at its two ends', async ({
    page,
    request,
  }) => {
    for (const child of ['Pharos', 'Lamp', 'Mirror']) {
      await saveConnection(request, {
        first: notes['Headland']!,
        second: notes[child]!,
        role: 'ROLE_CHILD',
        firstSide: { label: `holds ${child}` },
      })
    }
    await saveConnection(request, {
      first: notes['Lamp']!,
      second: notes['Mirror']!,
      role: 'ROLE_JUMP',
      firstSide: { label: 'lights', marker: 'MARKER_ARROW' },
      secondSide: { label: 'lit by', marker: 'MARKER_ARROW' },
    })

    await showInPlex(page, notes['Pharos']!)
    await expect(getGraph(page).getByRole('button', { name: 'Mirror, sibling' })).toBeAttached()

    // Each record is by the note its head is at: Lamp wrote 'lights' pointing at Mirror, Mirror wrote 'lit by' pointing at Lamp.
    await expect
      .poll(() =>
        isSetAtBothEnds(page, ['Lamp', 'Mirror'], { toFirst: 'lit by', toSecond: 'lights' }),
      )
      .toBe(true)
  })
})
