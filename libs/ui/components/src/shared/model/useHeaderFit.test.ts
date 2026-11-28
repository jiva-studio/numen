/** The level of a card header: the fullest one that is not crowded, found once for each width and never in a loop. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, ref, watch } from 'vue'
import { describe, expect, it } from 'vitest'
import type { Size } from '../lib/math/geometry'
import type { Viewport } from '../api/viewport'
import { useHeaderFit } from './useHeaderFit'

type Level = ReturnType<typeof useHeaderFit>['value']

const NEEDS: Record<Level, number> = { full: 300, icons: 150, menu: 40 }

const mountFit = (width: number) => {
  const header = document.createElement('div')
  const visited: Level[] = []
  let told: (size: Size) => void = () => undefined
  let isWatching = false
  let level!: ReturnType<typeof useHeaderFit>
  const viewport: Pick<Viewport, 'watch'> = {
    watch: (_of, took) => {
      told = took
      isWatching = true
      return () => {
        isWatching = false
      }
    },
  }
  const room = ref(width)
  Object.defineProperty(header, 'clientWidth', { get: () => room.value })
  Object.defineProperty(header, 'scrollWidth', { get: () => NEEDS[level.value] })
  const wrapper = mount(
    defineComponent({
      setup() {
        level = useHeaderFit(ref(header), viewport)
        watch(level, (next) => visited.push(next))
        return () => h('div')
      },
    }),
  )
  const resize = async (next: number): Promise<void> => {
    room.value = next
    told({ width: next, height: 20 })
    await flushPromises()
  }
  return { level, visited, resize, wrapper, isWatching: () => isWatching }
}

describe('useHeaderFit', () => {
  it('takes the fullest level that is not crowded', async () => {
    const { level } = mountFit(200)
    await flushPromises()
    expect(level.value).toBe('icons')
  })

  it('does not fit again when the header keeps its width', async () => {
    const { level, visited, resize } = mountFit(200)
    await resize(200)
    visited.length = 0
    await resize(200)
    expect(visited).toStrictEqual([])
    expect(level.value).toBe('icons')
  })

  it('fits again from the fullest level when the width changes', async () => {
    const { level, resize } = mountFit(200)
    await resize(400)
    expect(level.value).toBe('full')
    await resize(100)
    expect(level.value).toBe('menu')
  })

  it('ends at the level of the last width when widths come while it fits', async () => {
    const { level, resize } = mountFit(200)
    const first = resize(400)
    await resize(100)
    await first
    expect(level.value).toBe('menu')
  })

  it('stops watching and changes nothing after the header unmounts', async () => {
    const { level, resize, wrapper, isWatching } = mountFit(200)
    await flushPromises()
    wrapper.unmount()
    expect(isWatching()).toBe(false)
    await resize(400)
    expect(level.value).toBe('icons')
  })
})
