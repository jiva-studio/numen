/**
 * What the corner announces, and when. The regions are emptied and filled one tick
 * apart, so a screen reader announces each change.
 */
import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, shallowRef, watch } from 'vue'
import { useAnnouncer } from './announcer'
import type { Notice } from '../lib/notice'

const createQuietNotice = (id: string, text: string, about?: string): Notice => ({
  id,
  text,
  ...(about ? { about } : {}),
})
const loud = (id: string, text: string): Notice => ({ id, text, severity: 'error' })

/** The announcer mounted over a list of notices the test replaces. */
function listen(initial: readonly Notice[]) {
  const notices = shallowRef<readonly Notice[]>(initial)
  const next = shallowRef<readonly Notice[] | undefined>(undefined)
  let regions!: ReturnType<typeof useAnnouncer>
  const wrapper = mount(
    defineComponent({
      setup() {
        regions = useAnnouncer(() => notices.value)
        // Replaces the notices again inside the flush that announced the first set.
        watch(notices, () => {
          const queued = next.value
          next.value = undefined
          if (queued) notices.value = queued
        })
        return () => h('div')
      },
    }),
  )
  return { wrapper, notices, next, regions }
}

describe('what the corner reads out', () => {
  it('has nothing to say until the notices are drawn', async () => {
    const { regions } = listen([createQuietNotice('a', 'Renamed')])

    expect(regions.polite.value).toBe('')
    expect(regions.assertive.value).toBe('')

    await flushPromises()

    expect(regions.polite.value).toBe('Renamed')
    expect(regions.assertive.value).toBe('')
  })

  it('reads what a card is about after its words', async () => {
    const { regions } = listen([createQuietNotice('a', 'Renamed', 'notes.md')])

    await flushPromises()

    expect(regions.polite.value).toBe('Renamed — notes.md')
  })

  it('reads trouble over everything and the rest politely', async () => {
    const { regions } = listen([createQuietNotice('a', 'Renamed'), loud('b', 'Filed there')])

    await flushPromises()

    expect(regions.polite.value).toBe('Renamed')
    expect(regions.assertive.value).toBe('Filed there')
  })

  it('reads only trouble when there is nothing else', async () => {
    const { regions } = listen([loud('b', 'Filed there')])

    await flushPromises()

    expect(regions.polite.value).toBe('')
    expect(regions.assertive.value).toBe('Filed there')
  })

  it('reads several cards in one sentence each, joined', async () => {
    const { regions } = listen([
      createQuietNotice('a', 'Renamed'),
      createQuietNotice('b', 'Moved'),
      loud('c', 'Filed there'),
      loud('d', 'Locked'),
    ])

    await flushPromises()

    expect(regions.polite.value).toBe('Renamed. Moved')
    expect(regions.assertive.value).toBe('Filed there. Locked')
  })

  it('reads a card that arrives, and only that card', async () => {
    const { regions, notices } = listen([createQuietNotice('a', 'Renamed')])
    await flushPromises()

    notices.value = [createQuietNotice('a', 'Renamed'), createQuietNotice('b', 'Moved')]
    await flushPromises()

    expect(regions.polite.value).toBe('Moved')
    expect(regions.assertive.value).toBe('')
  })

  it('reads a card again when its words change', async () => {
    const { regions, notices } = listen([createQuietNotice('a', 'Indexing', '1 of 3')])
    await flushPromises()

    notices.value = [createQuietNotice('a', 'Indexing', '2 of 3')]
    await flushPromises()

    expect(regions.polite.value).toBe('Indexing — 2 of 3')
  })

  it('does not read a card again when its words have not changed', async () => {
    const { regions, notices } = listen([createQuietNotice('a', 'Renamed')])
    await flushPromises()

    notices.value = [createQuietNotice('a', 'Renamed')]
    await nextTick()

    expect(regions.polite.value).toBe('Renamed')
  })

  it('does not read a card that has gone', async () => {
    const { regions, notices } = listen([createQuietNotice('a', 'Renamed')])
    await flushPromises()

    notices.value = []
    await nextTick()

    expect(regions.polite.value).toBe('Renamed')
  })

  it('reads a card again when it returns after going', async () => {
    const { regions, notices } = listen([createQuietNotice('a', 'Renamed')])
    await flushPromises()
    notices.value = []
    await flushPromises()

    notices.value = [createQuietNotice('a', 'Renamed')]
    await nextTick()

    expect(regions.polite.value).toBe('')
    await flushPromises()
    expect(regions.polite.value).toBe('Renamed')
  })

  it('empties both regions before it fills them again', async () => {
    const { regions, notices } = listen([
      createQuietNotice('a', 'Renamed'),
      loud('b', 'Filed there'),
    ])
    await flushPromises()

    notices.value = [
      createQuietNotice('a', 'Renamed'),
      loud('b', 'Filed there'),
      createQuietNotice('c', 'Moved'),
    ]
    await nextTick()

    expect(regions.polite.value).toBe('')
    expect(regions.assertive.value).toBe('')

    await flushPromises()

    expect(regions.polite.value).toBe('Moved')
    expect(regions.assertive.value).toBe('')
  })

  it('reads together what two changes inside one tick brought', async () => {
    const { regions, notices, next } = listen([])
    await flushPromises()

    next.value = [createQuietNotice('a', 'Renamed'), createQuietNotice('b', 'Moved')]
    notices.value = [createQuietNotice('a', 'Renamed')]
    await flushPromises()

    expect(regions.polite.value).toBe('Renamed. Moved')
  })
})
