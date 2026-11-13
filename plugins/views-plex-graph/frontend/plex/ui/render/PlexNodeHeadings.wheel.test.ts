/** A wheel over the headings belongs to them while there is room to scroll, however small each step of it is. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexNodeHeadings from './PlexNodeHeadings.vue'
import { getNodeHeadings } from '../../lib/inside'
import type { PlacedNode } from '../../lib/node'
import type { PlexHeading } from '../../../types'

const MOST = 4
const HEIGHT = 20

const node: PlacedNode = {
  id: 'one',
  title: 'A node',
  relation: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

const headings: PlexHeading[] = Array.from({ length: MOST + 3 }, (_, index) => ({
  id: `${index}`,
  text: `Heading ${index}`,
  level: 1,
}))

const mountHeadings = () =>
  mount(PlexNodeHeadings, {
    props: {
      open: 1,
      hung: getNodeHeadings(node, headings, {
        options: { headingHeight: HEIGHT, headingIndent: 10, maxHeadings: MOST },
        deps: { viewport: { width: 1000, height: 600 }, margin: 20 },
      })!,
    },
  })

const wheel = (view: ReturnType<typeof mountHeadings>, deltaY: number) => {
  const event = new WheelEvent('wheel', { deltaY, deltaMode: 0, bubbles: true, cancelable: true })
  view.get('.plex__inside').element.dispatchEvent(event)
  return event
}

describe('the wheel over the headings', () => {
  it('is kept from the pane behind when a step is smaller than a heading and there is room below', () => {
    const view = mountHeadings()

    const event = wheel(view, 3)

    expect(event.defaultPrevented).toBe(true)
  })

  it('adds small steps up to a heading and scrolls it', async () => {
    const view = mountHeadings()
    const first = () => view.findAll('.plex__heading')[0]!.text()

    for (let step = 0; step < 7; step++) wheel(view, 3)
    await view.vm.$nextTick()

    expect(first()).toBe('Heading 1')
  })

  it('is left to the pane behind when there is no room above', () => {
    const view = mountHeadings()

    expect(wheel(view, -3).defaultPrevented).toBe(false)
  })

  it('is left to the pane behind when a step of nothing is made', () => {
    const view = mountHeadings()

    expect(wheel(view, 0).defaultPrevented).toBe(false)
  })
})
