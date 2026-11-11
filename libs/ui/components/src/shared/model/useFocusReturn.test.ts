import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { useFocusReturn } from './useFocusReturn'

const mountReturning = (origin: HTMLElement | null) =>
  mount(
    defineComponent({
      setup() {
        useFocusReturn(() => origin)
        return () => h('div')
      },
    }),
  )

describe('returning focus to the origin of a view', () => {
  it('focuses the origin when the view is unmounted', () => {
    const origin = document.createElement('button')
    document.body.append(origin)

    mountReturning(origin).unmount()

    expect(document.activeElement).toBe(origin)
    origin.remove()
  })

  it('leaves the focus alone when the origin left the page', () => {
    const origin = document.createElement('button')
    const view = mountReturning(origin)

    view.unmount()

    expect(document.activeElement).not.toBe(origin)
  })

  it('does nothing without an origin', () => {
    expect(() => mountReturning(null).unmount()).not.toThrow()
  })
})
