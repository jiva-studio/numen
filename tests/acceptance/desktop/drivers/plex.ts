/** The person at the plex: the notes drawn around the note in focus, and the links made from there. */
import { expect, type Locator } from 'playwright/test'
import { getGraph } from '../../../e2e/desktop/tests/support/workspace.js'
import { escapeRegExp, getStem, parseReference } from './rules.js'
import type { Scenario } from './scenario.js'

export class PlexDriver {
  constructor(private readonly scenario: Scenario) {}

  private getTitle(reference: string, name: string): string {
    return parseReference(reference, name).reference === 'at' ? getStem(name) : name
  }

  private getNote(title: string, relation: string): Locator {
    const role = relation === 'focus' ? 'img' : 'button'
    return getGraph(this.scenario.page).getByRole(role, { name: `${title}, ${relation}`, exact: true })
  }

  async expectShowsNote(title: string, relation: string, has: boolean): Promise<void> {
    await expect(this.getNote(title, relation)).toHaveCount(has ? 1 : 0)
  }

  private getNode(title: string): Locator {
    return getGraph(this.scenario.page).locator(`[aria-label^="${title}, "]`)
  }

  async hoverOver(title: string): Promise<void> {
    const node = this.getNode(title)
    await expect(node, `the plex draws no note titled "${title}"`).toHaveCount(1)
    await node.hover()
  }

  async expectShowsHeading(heading: string, title: string, has: boolean): Promise<void> {
    const headings = this.getNode(title).locator('.plex__heading-text').filter({ hasText: heading })
    await expect(headings).toHaveCount(has ? 1 : 0)
  }

  async choose(reference: string, name: string): Promise<void> {
    const title = escapeRegExp(this.getTitle(reference, name))
    await getGraph(this.scenario.page)
      .getByRole('button', { name: new RegExp(`^${title}, `) })
      .click()
  }

  async link(reference: string, name: string, toReference: string, toName: string): Promise<void> {
    const { page } = this.scenario
    const focus = this.getNote(this.getTitle(reference, name), 'focus')
    await focus.hover()
    const handle = page.locator('.plex__handle').first()
    await handle.hover()
    await handle.click()
    const field = page.getByRole('dialog', { name: 'New thought' }).getByRole('combobox')
    const toTitle = this.getTitle(toReference, toName)
    await field.fill(toTitle)
    await page.getByRole('option', { name: toTitle, exact: true }).first().click()
  }

  async expectShowsNotes(rows: readonly (readonly string[])[], has: boolean): Promise<void> {
    for (const [relation, title] of getTableBody(rows, ['relation', 'title'])) await this.expectShowsNote(title, relation, has)
  }

  /** The names under a note: the buttons of the plex whose nearest note is that note. */
  private async listNames(): Promise<{ name: string; owner: string }[]> {
    return getGraph(this.scenario.page)
      .locator('.plex__name')
      .evaluateAll((names) => {
        const nodes = [...document.querySelectorAll('.plex__node')]
        const distance = (name: DOMRect, node: DOMRect): number => {
          const dx = Math.max(node.left - (name.left + name.width / 2), 0, name.left + name.width / 2 - node.right)
          const dy = Math.max(node.top - (name.top + name.height / 2), 0, name.top + name.height / 2 - node.bottom)
          return Math.hypot(dx, dy)
        }
        return names.map((element) => {
          const box = element.getBoundingClientRect()
          const nearest = nodes.reduce((best, node) =>
            distance(box, node.getBoundingClientRect()) < distance(box, best.getBoundingClientRect()) ? node : best,
          )
          return { name: element.getAttribute('aria-label') ?? '', owner: (nearest.getAttribute('aria-label') ?? '').split(', ')[0] }
        })
      })
  }

  async expectShowsNames(rows: readonly (readonly string[])[]): Promise<void> {
    for (const [title, name] of getTableBody(rows, ['note', 'name'])) {
      await expect.poll(async () => (await this.listNames()).some((one) => one.owner === title && one.name === name), { message: `the plex shows no name "${name}" under "${title}"` }).toBe(true)
    }
  }

  async expectShowsNoNames(title: string): Promise<void> {
    await expect.poll(async () => (await this.listNames()).filter((one) => one.owner === title).map((one) => one.name)).toStrictEqual([])
  }

  async chooseName(name: string, title: string): Promise<void> {
    await expect.poll(async () => (await this.listNames()).some((one) => one.owner === title && one.name === name)).toBe(true)
    const index = (await this.listNames()).findIndex((one) => one.owner === title && one.name === name)
    await getGraph(this.scenario.page).locator('.plex__name').nth(index).click()
  }

  /** Drags from the handle of a note to the side where the relation is drawn, and picks the note in the popover that opens there. */
  async dragLink(title: string, relation: string, target: { title: string; isNew: boolean }): Promise<void> {
    const { page } = this.scenario
    const node = this.getNode(title)
    await node.hover()
    const handle = await getCenter(node.locator('.plex__handle'))
    const middle = await getCenter(node)
    const [dx, dy] = getSide(relation)
    await page.mouse.move(handle.x, handle.y)
    await page.mouse.down()
    await page.mouse.move(middle.x + dx * DRAG_LENGTH, middle.y + dy * DRAG_LENGTH, { steps: 12 })
    await page.mouse.up()
    const dialog = page.getByRole('dialog', { name: 'New thought' })
    await dialog.getByRole('combobox').fill(target.title)
    const option = target.isNew ? `Create "${target.title}"` : target.title
    await dialog.getByRole('option', { name: option, exact: true }).first().click()
  }

  /** The point in the middle of the line between two notes, where a click opens the inspector of their link. */
  private async findEdge(first: string, second: string): Promise<{ x: number; y: number } | null> {
    return getGraph(this.scenario.page).evaluate(
      (svg, titles) => {
        const PAD = 24
        const rects = titles.map((title) =>
          [...svg.querySelectorAll('.plex__node')]
            .find((node) => (node.getAttribute('aria-label') ?? '').startsWith(`${title}, `))
            ?.getBoundingClientRect(),
        )
        const [one, other] = rects
        if (!one || !other) return null
        const isNear = (point: DOMPoint, rect: DOMRect): boolean =>
          point.x >= rect.left - PAD && point.x <= rect.right + PAD && point.y >= rect.top - PAD && point.y <= rect.bottom + PAD
        for (const line of svg.querySelectorAll<SVGPathElement>('.plex__edge-hit')) {
          const matrix = line.getScreenCTM()
          if (!matrix) continue
          const length = line.getTotalLength()
          const at = (distance: number): DOMPoint => {
            const point = line.getPointAtLength(distance)
            return new DOMPoint(point.x, point.y).matrixTransform(matrix)
          }
          const [start, middle, end] = [at(0), at(length / 2), at(length)]
          if ((isNear(start, one) && isNear(end, other)) || (isNear(start, other) && isNear(end, one))) return { x: middle.x, y: middle.y }
        }
        return null
      },
      [first, second],
    )
  }

  private async openInspector(first: string, second: string): Promise<Locator> {
    const { page } = this.scenario
    const inspector = page.locator('.plex-link-popover')
    await expect(async () => {
      const at = await this.findEdge(first, second)
      if (at === null) throw new Error(`the plex draws no line between "${first}" and "${second}"`)
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

  private async fillRow(row: Locator, fields: readonly (readonly string[])[]): Promise<void> {
    for (const [field, value] of fields) {
      if (field === 'Description') await row.getByLabel('Link description').fill(value)
      else throw new Error(`the link inspector has no field "${field}": it has Description`)
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

  async addReverseLink(from: string, to: string, fields: readonly (readonly string[])[]): Promise<void> {
    const inspector = await this.openInspector(from, to)
    await inspector.getByRole('button', { name: /Add reverse direction/ }).click()
    await this.fillRow(await this.getRow(inspector, from), fields)
    await this.finish(inspector)
  }

  async removeLink(from: string, to: string): Promise<void> {
    const inspector = await this.openInspector(from, to)
    await (await this.getRow(inspector, from)).getByRole('button', { name: 'Remove link' }).click()
    await this.finish(inspector)
  }

  async removeConnection(first: string, second: string): Promise<void> {
    const inspector = await this.openInspector(first, second)
    const trashes = inspector.getByRole('button', { name: 'Remove link' })
    while ((await trashes.count()) > 0) await trashes.first().click()
    await expect(inspector.getByText('The link between these notes will be removed.')).toBeVisible()
    await this.finish(inspector)
  }
}

/** How far from the note a line is dragged, in pixels. */
const DRAG_LENGTH = 180

/** Where the relation is drawn, as a step in x and in y: parents above, children below, jumps on the left. */
const getSide = (relation: string): [number, number] => {
  const sides: Partial<Record<string, [number, number]>> = { parent: [0, -1], child: [0, 1], jump: [-1, 0] }
  const side = sides[relation]
  if (side === undefined) throw new Error(`a line cannot be dragged as a ${relation}: use parent, child or jump`)
  return side
}

const getCenter = async (element: Locator): Promise<{ x: number; y: number }> => {
  const box = await element.boundingBox()
  if (box === null) throw new Error('the plex does not draw the note')
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
}

/** The rows under the header of a table, which must name these columns. */
const getTableBody = (rows: readonly (readonly string[])[], columns: readonly string[]): readonly (readonly string[])[] => {
  if (rows[0]?.join('|') !== columns.join('|')) throw new Error(`the table must start with the header "| ${columns.join(' | ')} |"`)
  return rows.slice(1)
}
