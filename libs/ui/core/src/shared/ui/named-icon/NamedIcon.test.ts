import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createNamedIcon } from './create'
import { findNamedIcon, getIconKey } from './icons'
import NamedIcon from './NamedIcon.vue'

describe('getIconKey', () => {
  it('writes the words of a name joined', () => {
    expect(getIconKey('trash-2')).toBe('Trash2')
    expect(getIconKey('audio-lines')).toBe('AudioLines')
    expect(getIconKey('a-arrow-down')).toBe('AArrowDown')
  })
})

describe('findNamedIcon', () => {
  it('finds any icon of the set by its name', async () => {
    expect(await findNamedIcon('trash-2')).not.toBeNull()
    expect(await findNamedIcon('scan-text')).not.toBeNull()
    expect(await findNamedIcon('git-branch')).not.toBeNull()
  })

  it('finds nothing for a name with no icon or an empty name', async () => {
    expect(await findNamedIcon('no-such-icon')).toBeNull()
    expect(await findNamedIcon('')).toBeNull()
    expect(await findNamedIcon('icon')).toBeNull()
    expect(await findNamedIcon('use-lucide-props')).toBeNull()
  })
})

describe('NamedIcon', () => {
  it('draws the icon once it is found, with the attributes it was given', async () => {
    const icon = mount(NamedIcon, { props: { name: 'trash-2' }, attrs: { class: 'mark' } })

    expect(icon.find('svg').exists()).toBe(false)
    await flushPromises()

    expect(icon.find('svg.mark').exists()).toBe(true)
    expect(icon.find('svg').attributes('aria-hidden')).toBe('true')
  })

  it('draws nothing for a name with no icon', async () => {
    const icon = mount(NamedIcon, { props: { name: 'no-such-icon' } })
    await flushPromises()

    expect(icon.find('svg').exists()).toBe(false)
  })

  it('draws the icon of a new name when the name changes', async () => {
    const icon = mount(NamedIcon, { props: { name: 'trash-2' } })
    await flushPromises()
    const before = icon.html()

    await icon.setProps({ name: 'spell-check' })
    await flushPromises()

    expect(icon.find('svg').exists()).toBe(true)
    expect(icon.html()).not.toBe(before)
  })
})

describe('createNamedIcon', () => {
  it('gives the same component for the same name, and draws the icon', async () => {
    expect(createNamedIcon('trash-2')).toBe(createNamedIcon('trash-2'))
    expect(createNamedIcon('trash-2')).not.toBe(createNamedIcon('spell-check'))

    const icon = mount(createNamedIcon('trash-2'), { attrs: { class: 'mark' } })
    await flushPromises()

    expect(icon.find('svg.mark').exists()).toBe(true)
  })
})
