/**
 * The heading one section of a deck stands under: what is drawn on the rule,
 * what a name typed over it comes to, and what it is pressed to be rid of.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Band from './Band.vue'
import type { Band as Section } from './deck'

const BAND: Section = { id: 'roots', name: 'Roots', at: 1 }

const mountBand = (props: Record<string, unknown> = {}) =>
  mount(Band, { attachTo: document.body, props: { band: BAND, ...props } })

type Held = ReturnType<typeof mountBand>

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

describe('Band', () => {
  it('draws the section on a rule, which is what divides one from the next', () => {
    const held = mountBand()
    expect(held.get('.rule').find('input').exists()).toBe(true)
  })

  it('stands the name the section carries in the box', () => {
    expect(boxIn(mountBand()).element.value).toBe('Roots')
  })

  it('announces the box by the place the section stands among them', () => {
    expect(boxIn(mountBand()).attributes('aria-label')).toBe('Section 1')
  })

  it('emits the name typed over it, committed', async () => {
    const held = mountBand()
    await type(held, 'Leaves')
    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Leaves']])
  })

  it('renames nothing while the name is only being typed', async () => {
    const held = mountBand()
    await type(held, 'Leaves')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('takes a name another section carries, two sections being free to share one', async () => {
    const held = mountBand({ band: { id: 'shoots', name: 'Shoots', at: 2 } })
    await type(held, 'Roots')
    expect(held.find('[role="alert"]').exists()).toBe(false)
    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toEqual([['Roots']])
  })

  it('refuses a name with nothing in it, and says why to the box', async () => {
    const held = mountBand()
    await type(held, '   ')

    const said = held.get('[role="alert"]')
    expect(said.text()).toBe('A section needs a name')
    expect(boxIn(held).attributes('aria-invalid')).toBe('true')
    expect(boxIn(held).attributes('aria-describedby')).toBe(said.attributes('id'))

    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('abandons what was typed on escape, and keeps the name the section carries', async () => {
    const held = mountBand()
    await type(held, 'Leaves')
    await boxIn(held).trigger('keydown', { key: 'Escape' })
    await boxIn(held).trigger('change')
    expect(held.emitted('rename')).toBeUndefined()
  })

  it('emits the section asked to go, named to a reader', async () => {
    const held = mountBand()
    const away = held.get('.deed')
    expect(away.attributes('aria-label')).toBe('Remove: Roots')
    await away.trigger('click')
    expect(held.emitted('remove')).toEqual([[]])
  })
})
