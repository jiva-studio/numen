/** The person at the plex: the notes drawn around the note in focus, and the links made from there. */
import { expect, type Locator } from 'playwright/test'
import { getGraph } from '../../../e2e/desktop/tests/support/workspace.js'
import {
  escapeRegExp,
  getDirectionStep,
  getStem,
  getTableBody,
  isArrow,
  isOnSides,
  MAX_ARROW_CLICKS,
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

  /** The graph of the Plex tab, which the person looks at: another tab (the settings) may be in front of it. */
  private async show(): Promise<Locator> {
    const graph = getGraph(this.scenario.page)
    if (!(await graph.isVisible())) await this.scenario.page.locator('[role="tab"][data-workspace-tab^="plex"]').first().click()
    await expect(graph).toBeVisible()
    return graph
  }

  private async getNote(title: string, relation: string): Promise<Locator> {
    const role = relation === 'focus' ? 'img' : 'button'
    return (await this.show()).getByRole(role, { name: `${title}, ${relation}`, exact: true })
  }

  async expectShowsNote(title: string, relation: string, has: boolean): Promise<void> {
    await expect(await this.getNote(title, relation)).toHaveCount(has ? 1 : 0)
  }

  private async getNode(title: string): Promise<Locator> {
    return (await this.show()).locator(`.plex__node[aria-label^="${title}, "]`)
  }

  private async findNode(title: string): Promise<Locator> {
    const node = await this.getNode(title)
    await expect(node, `the plex draws no note titled "${title}"`).toHaveCount(1)
    return node
  }

  async restPointerOn(title: string): Promise<void> {
    await (await this.findNode(title)).hover()
  }

  async expectShowsHeading(heading: string, title: string, has: boolean): Promise<void> {
    const headings = (await this.getNode(title)).locator('.plex__heading-text').filter({ hasText: heading })
    await expect(headings).toHaveCount(has ? 1 : 0)
  }

  async enterHeading(heading: string, title: string): Promise<void> {
    const node = await this.findNode(title)
    await node.hover()
    await node.locator('.plex__heading').filter({ hasText: heading }).click()
  }

  async choose(reference: string, name: string): Promise<void> {
    const title = escapeRegExp(this.getTitle(reference, name))
    await (await this.show()).getByRole('button', { name: new RegExp(`^${title}, (focus|parent|child|jump|reference|sibling)$`) }).click()
  }

  async expectShowsNotes(rows: readonly (readonly string[])[], has: boolean): Promise<void> {
    for (const [relation, title] of getTableBody(rows, ['relation', 'title'])) await this.expectShowsNote(title, relation, has)
  }

  /** The note is on every side of the focus note that the table names: the middle of its box is compared with the middle of the box of the focus. */
  async expectSides(rows: readonly (readonly string[])[]): Promise<void> {
    const graph = await this.show()
    const focus = graph.locator('.plex__node--focus')
    for (const [title, side] of getTableBody(rows, ['title', 'side'])) {
      const sides = parseSides(side)
      await expect
        .poll(async () => isOnSides(sides, await getMiddle(await this.findNode(title)), await getMiddle(focus)), {
          message: `the note "${title}" is not ${side} of the focus note`,
        })
        .toBe(true)
    }
  }

  async clickHandle(title: string, toTitle: string): Promise<void> {
    const node = await this.findNode(title)
    await node.hover()
    await node.locator('.plex__handle').click()
    await this.pick(toTitle, toTitle)
  }

  private async pick(search: string, option: string): Promise<void> {
    const dialog = this.scenario.page.getByRole('dialog', { name: 'New thought' })
    await dialog.getByRole('combobox').fill(search)
    await dialog.getByRole('option', { name: option, exact: true }).first().click()
  }

  /** Drags from the handle of a note in the direction of a relation, and picks or makes the note in the popover that opens there. */
  async dragLine(direction: string, title: string, target: { title: string; isNew: boolean }): Promise<void> {
    const node = await this.findNode(title)
    await node.hover()
    const [dx, dy] = getDirectionStep(direction)
    const handle = await getMiddle(node.locator('.plex__handle'))
    const middle = await getMiddle(node)
    await this.drag(handle, { x: middle.x + dx * DRAG_LENGTH, y: middle.y + dy * DRAG_LENGTH })
    await this.pick(target.title, target.isNew ? `Create "${target.title}"` : target.title)
  }

  /** Drags from the handle of a note onto another note of the plex, which is a link of the kind its place gives it. */
  async dragOnto(title: string, toTitle: string): Promise<void> {
    const node = await this.findNode(title)
    await node.hover()
    await this.drag(await getMiddle(node.locator('.plex__handle')), await getMiddle(await this.findNode(toTitle)))
  }

  private async drag(from: Point, to: Point): Promise<void> {
    const { page } = this.scenario
    await page.mouse.move(from.x, from.y)
    await page.mouse.down()
    await page.mouse.move(to.x, to.y, { steps: 12 })
    await page.mouse.up()
  }

  private async getName(name: string, title: string): Promise<Locator> {
    return (await this.show()).getByRole('button', { name: `${name}, name under ${title}`, exact: true })
  }

  async expectShowsNames(rows: readonly (readonly string[])[]): Promise<void> {
    for (const [title, name] of getTableBody(rows, ['note', 'name'])) await expect(await this.getName(name, title)).toHaveCount(1)
  }

  async expectShowsNoNames(title: string): Promise<void> {
    const names = (await this.show()).locator(`[role="button"][aria-label$=", name under ${title}"]`)
    await expect(names).toHaveCount(0)
  }

  async chooseName(name: string, title: string): Promise<void> {
    await (await this.getName(name, title)).click()
  }

  /** The line between two notes, as the plex names it: from one to the other with the arrows it has, either way round. */
  private async getLine(first: string, second: string): Promise<Locator> {
    const names = `${escapeRegExp(first)} to ${escapeRegExp(second)}|${escapeRegExp(second)} to ${escapeRegExp(first)}`
    return (await this.show()).getByRole('button', { name: new RegExp(`^Line from (${names})(, arrow to .+)*$`) })
  }

  async expectDrawsLine(from: string, to: string, arrow: string): Promise<void> {
    const line = (await this.show()).getByRole('button', { name: `Line from ${from} to ${to}, arrow to ${arrow}`, exact: true })
    await expect(line).toHaveCount(1)
  }

  async expectDrawsNoLine(first: string, second: string): Promise<void> {
    await expect(await this.getLine(first, second)).toHaveCount(0)
  }

  /** A point on the line, in the middle of its length, where a click opens the inspector of its link. */
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

  /** The row of the record a note writes: the inspector lists the record of its first note, then the record of the other. */
  private async getRow(inspector: Locator, from: string): Promise<Locator> {
    const titles = await inspector.locator('.plex-link-popover__node-name').allTextContents()
    const rows = inspector.locator('.plex-link-row')
    const count = await rows.count()
    if (count === 2) return rows.nth(titles[0] === from ? 0 : 1)
    if (count === 1 && titles[0] === from) return rows.first()
    throw new Error(`the link inspector of ${titles.join(' and ')} has ${count} rows: none is the link from "${from}"`)
  }

  /** Clicks the arrow button of a row until it shows the arrow, which takes at most one click for each arrow it can show. */
  private async setArrow(row: Locator, arrow: string): Promise<void> {
    const button = row.getByRole('button', { name: /Click for: / })
    for (let clicks = 0; clicks <= MAX_ARROW_CLICKS; clicks += 1) {
      const name = (await button.getAttribute('aria-label')) ?? ''
      if (isArrow(name, arrow)) return
      if (clicks < MAX_ARROW_CLICKS) await button.click()
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
    await expect(this.scenario.page.getByRole('menuitem').first()).toBeVisible()
  }

  async listMenuItems(): Promise<string[]> {
    const items = this.scenario.page.getByRole('menuitem')
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

/** How far from the note a line is dragged, in pixels. */
const DRAG_LENGTH = 180

const getMiddle = async (element: Locator): Promise<Point> => {
  const box = await element.boundingBox()
  if (box === null) throw new Error('the plex does not draw the note')
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
}
