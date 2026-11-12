import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createNamedIcon } from './create'
import { findNamedIcon, getIconKey, isIconKey } from './icons'
import { buttonVariants } from '../button/variants'
import NamedIcon from './NamedIcon.vue'

describe('getIconKey', () => {
  it('writes the words of a name joined', () => {
    expect(getIconKey('trash-2')).toBe('Trash2')
    expect(getIconKey('audio-lines')).toBe('AudioLines')
    expect(getIconKey('a-arrow-down')).toBe('AArrowDown')
    expect(isIconKey('Trash2')).toBe(true)
    expect(isIconKey('trash2')).toBe(false)
    expect(isIconKey('trashA')).toBe(false)
    expect(isIconKey('Icon')).toBe(false)
    expect(isIconKey('LucideTrash')).toBe(false)
    expect(buttonVariants()).toContain('bg-accent')
    expect(buttonVariants({ variant: 'outline', size: 'small' })).toContain('border-field-rule')
    expect(buttonVariants({ variant: 'danger', size: 'icon' })).toContain('border-alarm')
    expect(buttonVariants({ variant: 'ghost', size: 'icon-small' })).toContain('hover:bg-raised')
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
    expect(await findNamedIcon('lucide-trash-2')).toBeNull()
    expect(await findNamedIcon('1-trash')).toBeNull()
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

  it('draws nothing for an empty name', async () => {
    const icon = mount(NamedIcon, { props: { name: '' } })
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
    const comp = createNamedIcon('trash-2') as { name?: string; inheritAttrs?: boolean }
    expect(comp.name).toBe('NamedIconOf')
    expect(comp.inheritAttrs).toBe(false)
    expect(createNamedIcon('trash-2')).toBe(createNamedIcon('trash-2'))
    expect(createNamedIcon('trash-2')).not.toBe(createNamedIcon('spell-check'))

    const icon = mount(createNamedIcon('trash-2'), { attrs: { class: 'mark' } })
    await flushPromises()

    expect(icon.find('svg.mark').exists()).toBe(true)
  })
})
