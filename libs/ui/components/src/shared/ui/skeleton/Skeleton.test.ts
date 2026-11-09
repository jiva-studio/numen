/**
 * The shape a placeholder takes: the room it holds, and that it says nothing
 * to a screen reader.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Skeleton from './Skeleton.vue'

describe('the room a placeholder holds', () => {
  it('is the width of its line and one em high where it is given nothing', () => {
    const style = mount(Skeleton).element.style
    expect(style.inlineSize).toBe('100%')
    expect(style.blockSize).toBe('1em')
  })

  it('is the width and the height it is given', () => {
    const style = mount(Skeleton, { props: { wide: '12rem', high: '2rem' } }).element.style
    expect(style.inlineSize).toBe('12rem')
    expect(style.blockSize).toBe('2rem')
  })

  it('follows the size when the size changes', async () => {
    const skeleton = mount(Skeleton, { props: { wide: '4rem' } })
    await skeleton.setProps({ wide: '8rem' })
    expect(skeleton.element.style.inlineSize).toBe('8rem')
  })
})

describe('what it is drawn as', () => {
  it('is rounded to its own ends only where it is a pill', async () => {
    const skeleton = mount(Skeleton)
    expect(skeleton.classes()).not.toContain('skeleton--pill')

    await skeleton.setProps({ isPill: true })
    expect(skeleton.classes()).toContain('skeleton--pill')
  })

  it('is hidden from a screen reader, because it says a value is pending and nothing more', () => {
    expect(mount(Skeleton).attributes('aria-hidden')).toBe('true')
  })
})
