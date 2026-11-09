/**
 * What the index stats chip shows and opens. The popover is drawn at the end of the document, so
 * it is read off the document and not off the wrapper.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Stats } from './api/stats'
import StatsIndicator from './StatsIndicator.vue'

const STATS: Stats = {
  vectors: 12480,
  chunks: 13100,
  waiting: 620,
  sources: 84,
  kinds: [
    { kind: 'notes', sources: 70, chunks: 4100 },
    { kind: 'pdf', sources: 0, chunks: 0 },
  ],
  model: 'bge-m3',
  dimensions: 1024,
  profile: 'balanced',
  threads: 4,
  speed: 18.5,
  databaseBytes: 73400320,
  vectorBytes: 52428800,
}

const mounted: VueWrapper[] = []

afterEach(() => {
  for (const wrapper of mounted.splice(0)) wrapper.unmount()
  document.body.replaceChildren()
})

const draw = (stats: Stats | null, phase: 'loading' | 'ready' | 'unavailable' = 'ready') => {
  const container = document.body.appendChild(document.createElement('div'))
  mounted.push(
    mount(StatsIndicator, { props: { stats, phase, locale: 'en-US' }, attachTo: container }),
  )
  return container
}

const getTrigger = (container: HTMLElement): HTMLElement => {
  const trigger = container.querySelector<HTMLElement>('button[aria-haspopup="dialog"]')
  if (!trigger) throw new Error('the chip draws no trigger')
  return trigger
}

const getDialog = (): HTMLElement | null =>
  document.body.querySelector<HTMLElement>('[role="dialog"][aria-label="Index stats"]')

const press = async (element: HTMLElement): Promise<void> => {
  element.click()
  await flushPromises()
}

describe('the index stats indicator', () => {
  it('keeps the mark and the figure in their own cells in every state', () => {
    for (const stats of [STATS, { ...STATS, waiting: 0 }, null]) {
      const trigger = getTrigger(draw(stats, stats ? 'ready' : 'loading'))
      expect(trigger.querySelector('.index-stats__icon')).not.toBeNull()
      expect(trigger.querySelector('.index-stats__figure')).not.toBeNull()
    }
  })

  it('is out of the tab order', () => {
    expect(getTrigger(draw(STATS)).getAttribute('tabindex')).toBe('-1')
  })

  it('shows the vectors alone, with no end value, while chunks wait', () => {
    const trigger = getTrigger(draw(STATS))

    expect(trigger.textContent?.trim()).toBe('⬡12,480')
    expect(trigger.textContent).not.toContain('/')
    expect(trigger.getAttribute('aria-label')).toBe('12,480 vectors, indexing')
    expect(trigger.getAttribute('title')).toBe('12,480 vectors, indexing')
  })

  it('shows the vectors alone once the queue is empty', () => {
    const trigger = getTrigger(draw({ ...STATS, vectors: 13100, waiting: 0 }))

    expect(trigger.textContent?.trim()).toBe('⬡13,100')
    expect(trigger.getAttribute('aria-label')).toBe('13,100 vectors')
  })

  it('draws the mark and a dash before the first answer', () => {
    expect(getTrigger(draw(null, 'loading')).textContent?.trim()).toBe('⬡–')
  })

  it('shows a dash where the index has nothing to show', () => {
    expect(getTrigger(draw(null, 'unavailable')).textContent?.trim()).toBe('⬡–')
  })

  it('opens the popover on a press, with the rows, and closes it on Escape', async () => {
    const chip = draw(STATS)
    expect(getDialog()).toBeNull()

    await press(getTrigger(chip))

    expect(getTrigger(chip).getAttribute('aria-expanded')).toBe('true')
    const labels = [...(getDialog()?.querySelectorAll('.index-popover__label') ?? [])].map(
      (one) => one.textContent,
    )
    expect(labels).toContain('Vectors')
    expect(labels).toContain('Chunks')
    expect(labels).toContain('Notes')
    expect(labels).not.toContain('PDF')

    document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()

    expect(getDialog()).toBeNull()
  })

  it('keeps the popover open when the window is resized', async () => {
    const chip = draw(STATS)
    await press(getTrigger(chip))

    window.dispatchEvent(new Event('resize'))
    await flushPromises()

    expect(getDialog()).not.toBeNull()
  })

  it('closes the popover with its close button', async () => {
    const chip = draw(STATS)
    await press(getTrigger(chip))

    await press(getDialog()!.querySelector<HTMLElement>('button[aria-label="Close index stats"]')!)

    expect(getDialog()).toBeNull()
    expect(getTrigger(chip).getAttribute('aria-expanded')).toBe('false')
  })

  it('opens no dialog on a press before the first count', async () => {
    const chip = draw(null, 'loading')

    await press(getTrigger(chip))

    expect(document.body.querySelector('[role="dialog"]')).toBeNull()
    expect(getTrigger(chip).getAttribute('aria-expanded')).toBe('false')
  })
})
