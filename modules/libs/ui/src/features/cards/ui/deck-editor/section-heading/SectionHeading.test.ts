/**
 * The heading one section of a deck stands under: what is drawn on the rule,
 * what a name typed over it comes to, and what it is pressed to be rid of.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import SectionHeading from './SectionHeading.vue'
import type { PlacedSection } from '../../../lib/grid'

const SECTION: PlacedSection = { id: 'roots', name: 'Roots', at: 1 }

const mountHeading = (props: Record<string, unknown> = {}) =>
  mount(SectionHeading, { attachTo: document.body, props: { section: SECTION, ...props } })

type Held = ReturnType<typeof mountHeading>

const boxIn = (held: Held) => held.get<HTMLInputElement>('input')

/** A name typed into the box and not yet committed. */
const type = async (held: Held, name: string): Promise<void> => {
  const box = boxIn(held)
  box.element.value = name
  await box.trigger('input')
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SectionHeading', () => {
  it('draws the section on a rule, which is what divides one from the next', () => {
    const held = mountHeading()
    expect(held.get('.divider').find('input').exists()).toBe(true)
  })

  it('stands the name the section carries in the box', () => {
    expect(boxIn(mountHeading()).element.value).toBe('Roots')
  })

  it('announces the box by the place the section stands among them', () => {
    expect(boxIn(mountHeading()).attributes('aria-label')).toBe('Section 1')
  })

  it('emits the name typed over it, committed', async () => {
    const held = mountHeading()
    await type(held, 'Leaves')
    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Leaves']])
  })

  it('renames nothing while the name is only being typed', async () => {
    const held = mountHeading()
    await type(held, 'Leaves')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('takes a name another section carries, two sections being free to share one', async () => {
    const held = mountHeading({ section: { id: 'shoots', name: 'Shoots', at: 2 } })
    await type(held, 'Roots')
    expect(held.find('[role="alert"]').exists()).toBe(false)
    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Roots']])
  })

  it('refuses a name with nothing in it, and says why to the box', async () => {
    const held = mountHeading()
    await type(held, '   ')

    const said = held.get('[role="alert"]')
    expect(said.text()).toBe('A section needs a name')
    expect(boxIn(held).attributes('aria-invalid')).toBe('true')
    expect(boxIn(held).attributes('aria-describedby')).toBe(said.attributes('id'))

    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('abandons what was typed on escape, and keeps the name the section carries', async () => {
    const held = mountHeading()
    await type(held, 'Leaves')
    await boxIn(held).trigger('keydown', { key: 'Escape' })
    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('emits the section asked to go, named to a reader', async () => {
    const held = mountHeading()
    const away = held.get('.remove-button')
    expect(away.attributes('aria-label')).toBe('Remove: Section 1')
    await away.trigger('click')
    expect(held.emitted('remove')).toEqual([[]])
  })

  it('names the way to be rid of it by the place it stands, a bare heading and all', () => {
    const held = mountHeading({ section: { id: 'bare', name: '', at: 2 } })
    expect(held.get('.remove-button').attributes('aria-label')).toBe('Remove: Section 2')
  })
})
