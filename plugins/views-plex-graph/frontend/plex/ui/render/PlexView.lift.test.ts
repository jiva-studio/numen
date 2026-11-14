/**
 * The note that has opened is drawn over its neighbors from a layer of its own: the notes are never
 * reordered, so the one under the pointer keeps its place in the document, its keyboard focus and its pointer events.
 */
import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import PlexView from './PlexView.vue'
import { arrangePlex, DEFAULT_OPTIONS } from '../../lib/arrange'
import { DEFAULT_BEHAVIOR } from '../../model/behavior'
import { EXPANSION_DURATION } from '../../model/sizing/expansion'
import { stubClock } from '../../../fixtures/clock'
import type { PlexGraph } from '../../../types'

const WAIT = 500

const graph: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Start', relation: 'focus' },
    { id: 'below', title: 'Below', relation: 'child' },
    { id: 'aside', title: 'Aside', relation: 'jump' },
  ],
  edges: [
    { from: 'focus', to: 'below' },
    { from: 'focus', to: 'aside' },
  ],
}

afterEach(() => {
  vi.useRealTimers()
  document.body.replaceChildren()
})

const mountView = () => {
  vi.useFakeTimers()
  const world = stubClock()
  const view = mount(PlexView, {
    attachTo: document.body,
    props: {
      namesOpacity: 0.3,
      focusedNamesOpacity: 0.9,
      frame: arrangePlex(graph),
      viewport: { width: 1200, height: 800 },
      nodeSize: DEFAULT_OPTIONS.nodeSize,
      widen: (node) => (node.id === 'below' ? { width: 400, offset: 0 } : null),
      behavior: { ...DEFAULT_BEHAVIOR, expansionDelay: WAIT, clock: world.clock },
    },
  })
  return { view, world }
}

const getNoteGroups = (view: VueWrapper) =>
  view.findAll('.plex__stack > .plex__node').map((group) => group.element)

const rest = async (mounted: ReturnType<typeof mountView>, title = 'Below') => {
  await mounted.view.get(`[aria-label^="${title}"]`).trigger('pointerenter')
  await vi.advanceTimersByTimeAsync(WAIT)
  mounted.world.run()
  await mounted.view.vm.$nextTick()
}

describe('a note that has rested open', () => {
  it('keeps every note group where it was in the document', async () => {
    const mounted = mountView()
    const before = getNoteGroups(mounted.view)
    const root = mounted.view.element
    const observer = new MutationObserver(vi.fn())
    observer.observe(root, { childList: true, subtree: true })

    await rest(mounted)

    const moved = observer
      .takeRecords()
      .flatMap((record) => [...record.addedNodes, ...record.removedNodes])
      .filter((one) => one instanceof Element && one.matches('.plex__stack, .plex__node'))
    observer.disconnect()
    expect(moved).toHaveLength(0)
    const after = getNoteGroups(mounted.view)
    expect(after).toHaveLength(before.length)
    expect(after.every((group, index) => group === before[index])).toBe(true)
  })

  it('draws its box widened from a layer after every note group, and draws its title once', async () => {
    const mounted = mountView()
    await rest(mounted)

    const lifted = mounted.view.findAll('.plex__lifted')
    expect(lifted).toHaveLength(1)
    for (const group of getNoteGroups(mounted.view)) {
      expect(
        group.compareDocumentPosition(lifted[0]!.element) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy()
    }
    expect(lifted[0]!.get('.plex__box').attributes('width')).toBe('400')
    const titles = mounted.view.findAll('.plex__title-text').filter((one) => one.text() === 'Below')
    expect(titles).toHaveLength(1)
  })

  it('keeps the keyboard focus on the note group', async () => {
    const mounted = mountView()
    const group = mounted.view.get('[aria-label^="Below"]').element as SVGGElement
    group.focus()
    expect(group.ownerDocument.activeElement).toBe(group)

    await rest(mounted)

    expect(group.ownerDocument.activeElement).toBe(group)
  })

  it('stays open while the pointer moves from the note group onto its lifted box', async () => {
    const mounted = mountView()
    await rest(mounted)
    const group = mounted.view.get('[aria-label^="Below"]')
    const lifted = mounted.view.get('.plex__lifted')

    group.element.dispatchEvent(
      new PointerEvent('pointerleave', { bubbles: false, relatedTarget: lifted.element }),
    )
    await vi.advanceTimersByTimeAsync(WAIT)
    mounted.world.run(EXPANSION_DURATION)
    await mounted.view.vm.$nextTick()

    expect(mounted.view.findAll('.plex__lifted')).toHaveLength(1)
  })

  it('returns to its place once the pointer leaves it', async () => {
    const mounted = mountView()
    await rest(mounted)

    await mounted.view.get('.plex__lifted').trigger('pointerleave')
    mounted.world.run(1000)
    await mounted.view.vm.$nextTick()

    expect(mounted.view.find('.plex__lifted').exists()).toBe(false)
    expect(mounted.view.get('[aria-label^="Below"]').get('.plex__box').attributes('width')).toBe(
      `${DEFAULT_OPTIONS.nodeSize.width}`,
    )
  })

  it('reports a click on the lifted box as a click on the note', async () => {
    const mounted = mountView()
    await rest(mounted)

    await mounted.view.get('.plex__lifted').trigger('click')

    expect(mounted.view.emitted('activate')).toStrictEqual([['below']])
  })

  it('is never open for two notes at once', async () => {
    const mounted = mountView()
    await rest(mounted)

    await mounted.view.get('[aria-label^="Aside"]').trigger('pointerenter')
    await mounted.view.vm.$nextTick()
    mounted.world.run(1000)
    await mounted.view.vm.$nextTick()

    expect(mounted.view.findAll('.plex__lifted').length).toBeLessThanOrEqual(1)
  })
})
