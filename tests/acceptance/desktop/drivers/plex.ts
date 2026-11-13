/** The person at the plex: the notes drawn around the note in focus, and the links made from there. */
import { expect, type Locator } from 'playwright/test'
import { getGraph } from '../../../e2e/desktop/tests/support/workspace.js'
import {
  escapeRegExp,
  getDirectionStep,
  getStem,
  getTableBody,
  HEADINGS_OPEN_WAIT_MS,
  isArrow,
  isOnSides,
  MAX_ARROW_CLICKS,
  NO_ARROW,
  parseReference,
  parseSides,
  type Extent,
} from './rules.js'
import type { Scenario } from './scenario.js'

export class PlexDriver {
  constructor(private readonly scenario: Scenario) {}

  private getTitle(reference: string, name: string): string {
    return parseReference(reference, name).reference === 'at' ? getStem(name) : name
  }

  /** The plex of the window once it has drawn its focus note: a note is absent from a plex that has not drawn yet, so an absence is read after this. */
  private async getGraph(): Promise<Locator> {
    // The settings tab may be in front of the Plex tab, so the graph is looked up each time.
    const graph = getGraph(this.scenario.page)
    if (!(await graph.isVisible())) await this.scenario.page.locator('[role="tab"][data-workspace-tab^="plex"]').first().click()
    await expect(graph).toBeVisible()
    await expect(graph.locator('.plex__node--focus'), 'the plex draws no focus note').toHaveCount(1)
    return graph
  }

  private async getNote(title: string, relation: string): Promise<Locator> {
    const role = relation === 'focus' ? 'img' : 'button'
    return (await this.getGraph()).getByRole(role, { name: `${title}, ${relation}`, exact: true })
  }

  async expectShowsNote(title: string, relation: string, has: boolean): Promise<void> {
    await expect(await this.getNote(title, relation)).toHaveCount(has ? 1 : 0)
  }

  private async getNode(title: string): Promise<Locator> {
    const graph = await this.getGraph()
    return graph.locator('.plex__node').and(graph.getByLabel(new RegExp(`^${escapeRegExp(title)}, `)))
  }

  private async findNode(title: string): Promise<Locator> {
    const node = await this.getNode(title)
    await expect(node, `the plex draws no note titled "${title}"`).toHaveCount(1)
    return node
  }

  /** A part of a note (its pill, its handle, its headings): in the node, or in the layer over the notes once the note is open. */
  private async getPart(title: string, selector: string): Promise<Locator> {
    const graph = await this.getGraph()
    const lifted = graph
      .locator('.plex__lifted')
      .filter({ has: graph.locator('.plex__title-text').filter({ hasText: getWholeText(title) }) })
    return (await this.getNode(title)).locator(selector).or(lifted.locator(selector))
  }

  private async hoverNote(title: string): Promise<void> {
    await this.findNode(title)
    const at = await getMiddle(await this.getPart(title, '.plex__box'))
    await this.scenario.page.mouse.move(at.x, at.y)
  }

  async restPointerOn(title: string): Promise<void> {
    await this.hoverNote(title)
  }

  async expectShowsHeading(heading: string, title: string, has: boolean): Promise<void> {
    const headings = (await this.getPart(title, '.plex__heading-text')).filter({ hasText: getWholeText(heading) })
    if (has) {
      await expect(headings).toHaveCount(1)
      return
    }
    await this.expectStaysAbsent(headings)
  }

  /** The list of headings opens after the wait of the pointer: an absence is held for longer than that wait. */
  private async expectStaysAbsent(locator: Locator): Promise<void> {
    const deadline = Date.now() + HEADINGS_HOLD_MS
    do {
      await expect(locator).toHaveCount(0)
      await this.scenario.page.waitForTimeout(HOLD_STEP_MS)
    } while (Date.now() < deadline)
  }

  async enterHeading(heading: string, title: string): Promise<void> {
    await this.hoverNote(title)
    await (await this.getPart(title, '.plex__heading')).filter({ hasText: getWholeText(heading) }).click()
  }

  async choose(reference: string, name: string): Promise<void> {
    const title = escapeRegExp(this.getTitle(reference, name))
    await (await this.getGraph()).getByRole('button', { name: new RegExp(`^${title}, (focus|parent|child|jump|reference|sibling)$`) }).click()
  }

  async expectShowsNotes(rows: readonly (readonly string[])[], has: boolean): Promise<void> {
    for (const [relation, title] of getTableBody(rows, ['relation', 'title'])) await this.expectShowsNote(title, relation, has)
  }

  /** The note is on every side of the focus note that the table names: the middle of its box is compared with the middle of the box of the focus. */
  async expectSides(rows: readonly (readonly string[])[]): Promise<void> {
    const graph = await this.getGraph()
    const focus = graph.locator('.plex__node--focus')
    for (const [title, side] of getTableBody(rows, ['title', 'side'])) {
      const sides = parseSides(side)
      await expect
        .poll(async () => isOnSides(sides, await getMiddle(await this.getPart(title, '.plex__box')), await getMiddle(focus.locator('.plex__box'))), {
          message: `the note "${title}" is not ${side} of the focus note`,
        })
        .toBe(true)
    }
  }

  async clickHandle(title: string, toTitle: string): Promise<void> {
    await this.hoverNote(title)
    await (await this.waitForHandle(title)).click()
    await this.pick(toTitle, toTitle)
  }

  private async pick(search: string, option: string): Promise<void> {
    const dialog = this.scenario.page.getByRole('dialog', { name: 'New thought' })
    const options = dialog.getByRole('option', { name: option, exact: true })
    // The list is filled once the notes are searched, so the words are typed again until the option shows.
    await expect(async () => {
      await dialog.getByRole('combobox').fill('')
      await dialog.getByRole('combobox').fill(search)
      await expect(options.first()).toBeVisible({ timeout: 3_000 })
    }).toPass()
    await options.first().click()
  }

  /** The handle of a node, once it is shown and has stopped moving. */
  private async waitForHandle(title: string): Promise<Locator> {
    const handle = await this.getPart(title, '.plex__handle')
    await expect(handle).toBeVisible()
    let before = ''
    await expect
      .poll(async () => {
        const now = JSON.stringify(await handle.boundingBox())
        const isStill = now === before
        before = now
        return isStill
      })
      .toBe(true)
    return handle
  }

  /** Drags from the handle of a note in the direction of a relation, and picks or makes the note in the popover that opens there. */
  async dragLine(direction: string, title: string, target: { title: string; isNew: boolean }): Promise<void> {
    await this.hoverNote(title)
    const [dx, dy] = getDirectionStep(direction)
    const handle = await getMiddle(await this.waitForHandle(title))
    const middle = await getMiddle(await this.getPart(title, '.plex__box'))
    const length = Math.min(DRAG_LENGTH, (await this.measureRoom(middle, dx, dy)) - DRAG_MARGIN)
    const end = { x: middle.x + dx * length, y: middle.y + dy * length }
    await this.expectEmptyPlace(end)
    await this.drag(handle, end)
    await this.pick(target.title, target.isNew ? `Create "${target.title}"` : target.title)
  }

  /** Drags from the handle of a note onto another note of the plex, which is a link of the kind its place gives it. */
  async dragOnto(title: string, toTitle: string): Promise<void> {
    await this.hoverNote(title)
    const handle = await getMiddle(await this.waitForHandle(title))
    await this.drag(handle, await getMiddle(await this.getPart(toTitle, '.plex__box')))
  }

  /** How far the plex reaches from a point in a direction. */
  private async measureRoom(from: Point, dx: number, dy: number): Promise<number> {
    const area = await (await this.getGraph()).boundingBox()
    if (area === null) throw new Error('the plex is not drawn')
    if (dx < 0) return from.x - area.x
    if (dx > 0) return area.x + area.width - from.x
    return dy < 0 ? from.y - area.y : area.y + area.height - from.y
  }

  /** A line that ends on a note links it: the end of a line that makes a note is on no note. */
  private async expectEmptyPlace(at: Point): Promise<void> {
    const isOnNote = await this.scenario.page.evaluate(
      ([x, y]) => document.elementsFromPoint(x, y).some((one) => one.closest('.plex__node') !== null),
      [at.x, at.y] as const,
    )
    if (isOnNote) throw new Error(`the line would end on a note, at ${Math.round(at.x)} and ${Math.round(at.y)}: the plex has no empty place there`)
  }

  private async drag(from: Point, to: Point): Promise<void> {
    const { page } = this.scenario
    await page.mouse.move(from.x, from.y)
    await page.mouse.down()
    await page.mouse.move(to.x, to.y, { steps: 12 })
    await page.mouse.up()
  }

  private async getName(name: string, title: string): Promise<Locator> {
    return (await this.getGraph()).getByRole('button', { name: new RegExp(`^${escapeRegExp(name)}, name under ${escapeRegExp(title)}$`) })
  }

  async expectShowsNames(rows: readonly (readonly string[])[]): Promise<void> {
    for (const [title, name] of getTableBody(rows, ['note', 'name'])) await expect(await this.getName(name, title)).toHaveCount(1)
  }

  async expectShowsNoNames(title: string): Promise<void> {
    const nameButtons = (await this.getGraph()).getByRole('button', { name: new RegExp(`, name under ${escapeRegExp(title)}$`) })
    await expect(nameButtons).toHaveCount(0)
  }

  async chooseName(name: string, title: string): Promise<void> {
    await (await this.getName(name, title)).click()
  }

  private async getLine(first: string, second: string): Promise<Locator> {
    const names = `${escapeRegExp(first)} to ${escapeRegExp(second)}|${escapeRegExp(second)} to ${escapeRegExp(first)}`
    return (await this.getGraph()).getByRole('button', { name: new RegExp(`^Line from (${names})(, arrow to .+)*$`) })
  }

  /** The line joins the two notes, in either order; an arrow names the note it points to, `NO_ARROW` says it has none, and no arrow asks nothing of it. */
  async expectDrawsLine(first: string, second: string, arrow: string | undefined): Promise<void> {
    const pair = `${escapeRegExp(first)} to ${escapeRegExp(second)}|${escapeRegExp(second)} to ${escapeRegExp(first)}`
    const end = arrow === undefined ? '(, arrow to .+)?' : arrow === NO_ARROW ? '' : `, arrow to ${escapeRegExp(arrow)}`
    const line = (await this.getGraph()).getByRole('button', { name: new RegExp(`^Line from (${pair})${end}$`) })
    await expect(line).toHaveCount(1)
  }

  private async findMiddle(line: Locator): Promise<Point> {
    return line.evaluate((path: SVGPathElement) => {
      const matrix = path.getScreenCTM()
      if (!matrix) throw new Error('the plex does not draw the line')
      const point = path.getPointAtLength(path.getTotalLength() / 2)
      const at = new DOMPoint(point.x, point.y).matrixTransform(matrix)
      return { x: at.x, y: at.y }
    })
  }

  private async openInspector(first: string, second: string): Promise<Locator> {
    const { page } = this.scenario
    const inspector = page.locator('.plex-link-popover')
    const line = await this.getLine(first, second)
    await expect(line, `the plex draws no line between "${first}" and "${second}"`).toHaveCount(1)
    await expect(async () => {
      const at = await this.findMiddle(line)
      await page.mouse.click(at.x, at.y)
      await expect(inspector).toBeVisible({ timeout: 1_000 })
    }).toPass()
    return inspector
  }

  private async getRow(inspector: Locator, from: string): Promise<Locator> {
    const titles = await inspector.locator('.plex-link-popover__node-name').allTextContents()
    const rows = inspector.locator('.plex-link-row')
    const count = await rows.count()
    if (count === 2) return rows.nth(titles[0] === from ? 0 : 1)
    if (count === 1 && titles[0] === from) return rows.first()
    throw new Error(`the link inspector of ${titles.join(' and ')} has ${count} rows: none is the link from "${from}"`)
  }

  private async setArrow(row: Locator, arrow: string): Promise<void> {
    const button = row.getByRole('button', { name: /Click for: / })
    for (let clicks = 0; clicks <= MAX_ARROW_CLICKS; clicks += 1) {
      const name = (await button.getAttribute('aria-label')) ?? ''
      if (isArrow(name, arrow)) return
      if (clicks < MAX_ARROW_CLICKS) {
        await button.click()
        await expect(button).not.toHaveAttribute('aria-label', name)
      }
    }
    throw new Error(`the arrow button of the link does not show "${arrow}" after ${MAX_ARROW_CLICKS} clicks`)
  }

  private async fillRow(row: Locator, fields: readonly (readonly string[])[]): Promise<void> {
    for (const [field, value] of fields) {
      if (field === 'Description') await row.getByLabel('Link description').fill(value)
      else if (field === 'Arrow') await this.setArrow(row, value)
      else throw new Error(`the link inspector has no field "${field}": it has Arrow and Description`)
    }
  }

  private async finish(inspector: Locator): Promise<void> {
    await inspector.getByRole('button', { name: /^Done/ }).click()
    await expect(inspector).toBeHidden()
  }

  async changeLink(from: string, to: string, fields: readonly (readonly string[])[]): Promise<void> {
    const inspector = await this.openInspector(from, to)
    await this.fillRow(await this.getRow(inspector, from), fields)
    await this.finish(inspector)
  }

  async cancelLinkChange(from: string, to: string, fields: readonly (readonly string[])[]): Promise<void> {
    const inspector = await this.openInspector(from, to)
    await this.fillRow(await this.getRow(inspector, from), fields)
    await inspector.getByRole('button', { name: 'Cancel' }).click()
    await expect(inspector).toBeHidden()
  }

  async addReverseLink(from: string, to: string): Promise<void> {
    const inspector = await this.openInspector(from, to)
    await inspector.getByRole('button', { name: /Add reverse direction/ }).click()
    await expect(inspector.locator('.plex-link-row')).toHaveCount(2)
    await this.finish(inspector)
  }

  async removeLink({ kind, first, second }: Extent): Promise<void> {
    if (kind === 'connection') await this.removeConnection(first, second)
    else await this.removeDirection(first, second)
  }

  private async removeDirection(from: string, to: string): Promise<void> {
    const inspector = await this.openInspector(from, to)
    await (await this.getRow(inspector, from)).getByRole('button', { name: 'Remove link' }).click()
    await this.finish(inspector)
  }

  private async removeConnection(first: string, second: string): Promise<void> {
    const inspector = await this.openInspector(first, second)
    const trashes = inspector.getByRole('button', { name: 'Remove link' })
    while ((await trashes.count()) > 0) await trashes.first().click()
    await expect(inspector.getByText('The link between these notes will be removed.')).toBeVisible()
    await this.finish(inspector)
  }

  async openMenu(title: string): Promise<void> {
    await (await this.findNode(title)).click({ button: 'right' })
    await expect(this.scenario.page.getByRole('menu').getByRole('menuitem').first()).toBeVisible()
  }

  async chooseMenuItem(item: string): Promise<void> {
    const menuItem = this.scenario.page.getByRole('menu').getByRole('menuitem', { name: item, exact: true })
    await expect(menuItem, `the menu has no item "${item}"`).toHaveCount(1)
    await menuItem.click()
  }

  async expectShowsOnlyFocus(title: string): Promise<void> {
    const graph = await this.getGraph()
    await expect(graph.getByRole('img', { name: `${title}, focus`, exact: true })).toHaveCount(1)
    await expect(graph.locator('.plex__node[aria-label]')).toHaveCount(1)
  }

  async listMenuItems(): Promise<string[]> {
    const items = this.scenario.page.getByRole('menu').getByRole('menuitem')
    return (await items.allInnerTexts()).map((text) => text.trim())
  }

  async expectMenuItems(rows: readonly (readonly string[])[], rule: 'only' | 'include' | 'exclude'): Promise<void> {
    const wanted = getTableBody(rows, ['item']).map(([item]) => item)
    await expect
      .poll(async () => {
        const items = await this.listMenuItems()
        if (rule === 'only') return wanted.length === items.length && wanted.every((item) => items.includes(item))
        if (rule === 'include') return wanted.every((item) => items.includes(item))
        return wanted.every((item) => !items.includes(item))
      })
      .toBe(true)
  }
}

interface Point {
  x: number
  y: number
}

/** How far from the note a line is dragged at most, in pixels. */
const DRAG_LENGTH = 180

/** How much of the plex stays between the end of a dragged line and its edge. */
const DRAG_MARGIN = 24

const HEADINGS_HOLD_MS = 2 * HEADINGS_OPEN_WAIT_MS + 300
const HOLD_STEP_MS = 100

const getWholeText = (text: string): RegExp => new RegExp(`^${escapeRegExp(text)}$`)

const getMiddle = async (element: Locator): Promise<Point> => {
  const box = await element.boundingBox()
  if (box === null) throw new Error('the plex does not draw the note')
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
}
