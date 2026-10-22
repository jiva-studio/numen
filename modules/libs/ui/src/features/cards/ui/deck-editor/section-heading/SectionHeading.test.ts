/**
 * The heading one section of a deck is placed under: what is drawn on the rule,
 * what a name typed over it evaluates to, and what it is pressed to be rid of.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import SectionHeading from './SectionHeading.vue'
import type { PlacedSection } from '../../../lib/grid'

const SECTION: PlacedSection = { id: 'roots', name: 'Roots', at: 1 }

const mountHeading = (props: Record<string, unknown> = {}) =>
  mount(SectionHeading, { attachTo: document.body, props: { section: SECTION, ...props } })

type Held = ReturnType<typeof mountHeading>

const getBox = (wrapper: Held) => wrapper.get<HTMLInputElement>('input')

/** A name typed into the box and not yet committed. */
const type = async (wrapper: Held, name: string): Promise<void> => {
  const box = getBox(wrapper)
  box.element.value = name
  await box.trigger('input')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SectionHeading', () => {
  it('draws the section on a rule, which is what divides one from the next', () => {
    const target = mountHeading()
    expect(target.get('.divider').find('input').exists()).toBe(true)
  })

  it('remains the name the section carries in the box', () => {
    expect(getBox(mountHeading()).element.value).toBe('Roots')
  })

  it('announces the box by the place the section remains among them', () => {
    expect(getBox(mountHeading()).attributes('aria-label')).toBe('Section 1')
  })

  it('emits the name typed over it, committed', async () => {
    const target = mountHeading()
    await type(target, 'Leaves')
    await getBox(target).trigger('change')
    expect(target.emitted('rename')).toEqual([['Leaves']])
  })

  it('does not rename while the name is only being typed', async () => {
    const target = mountHeading()
    await type(target, 'Leaves')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('takes a name another section carries, two sections being free to share one', async () => {
    const target = mountHeading({ section: { id: 'shoots', name: 'Shoots', at: 2 } })
    await type(target, 'Roots')
    expect(target.find('[role="alert"]').exists()).toBe(false)
    await getBox(target).trigger('change')
    expect(target.emitted('rename')).toEqual([['Roots']])
  })

  it('refuses a name with nothing in it, and says why to the box', async () => {
    const target = mountHeading()
    await type(target, '   ')

    const response = target.get('[role="alert"]')
    expect(response.text()).toBe('A section needs a name')
    expect(getBox(target).attributes('aria-invalid')).toBe('true')
    expect(getBox(target).attributes('aria-describedby')).toBe(response.attributes('id'))

    await getBox(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('abandons what was typed on escape, and keeps the name the section carries', async () => {
    const target = mountHeading()
    await type(target, 'Leaves')
    await getBox(target).trigger('keydown', { key: 'Escape' })
    await getBox(target).trigger('change')
    expect(target.emitted('rename')).toBeUndefined()
  })

  it('emits the section requested to go, named to a reader', async () => {
    const target = mountHeading()
    const away = target.get('.remove-button')
    expect(away.attributes('aria-label')).toBe('Remove: Section 1')
    await away.trigger('click')
    expect(target.emitted('remove')).toEqual([[]])
  })

  it('names the way to be rid of it by the place it remains, a bare heading and all', () => {
    const target = mountHeading({ section: { id: 'bare', name: '', at: 2 } })
    expect(target.get('.remove-button').attributes('aria-label')).toBe('Remove: Section 2')
  })
})
