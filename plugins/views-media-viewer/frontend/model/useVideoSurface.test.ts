import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h, nextTick, ref, useTemplateRef } from 'vue'
import { useVideoSurface } from './useVideoSurface'

const mountSurface = (isShownAtFirst: boolean) => {
  const isShown = ref(isShownAtFirst)
  const log: string[] = []
  const view = mount(
    defineComponent({
      setup() {
        const container = useTemplateRef<HTMLElement>('container')
        useVideoSurface(
          container,
          () => isShown.value,
          () => ({
            attach: (element) => log.push(`attach ${element.tagName}`),
            release: (element) => log.push(`release ${element.tagName}`),
          }),
        )
        return () => h('div', { ref: 'container' })
      },
    }),
  )
  return { isShown, log, view }
}

describe('the surface a video is drawn in', () => {
  it('takes the media element while the view is shown', async () => {
    const { log } = mountSurface(true)
    await nextTick()

    expect(log).toEqual(['attach DIV'])
  })

  it('gives the media element back when the view stops being shown', async () => {
    const { isShown, log } = mountSurface(true)
    await nextTick()

    isShown.value = false
    await nextTick()

    expect(log).toEqual(['attach DIV', 'release DIV'])
  })

  it('gives the media element back when the view is unmounted', async () => {
    const { log, view } = mountSurface(true)
    await nextTick()

    view.unmount()

    expect(log).toEqual(['attach DIV', 'release DIV'])
  })
})
