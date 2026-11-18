/** The names of one side of a node, with the arrow of each relation drawn before the text. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import type { RelatedName } from '../types'
import RelatedNames from './RelatedNames.vue'

/** The icons of the shared set load on first use, slowly on a busy machine. */
const ICON_TIMEOUT_MS = 30_000

const createName = (
  title: string,
  relation: RelatedName['relation'],
  direction?: RelatedName['direction'],
): RelatedName => ({
  path: `notes/${title}.md`,
  title,
  relation,
  ...(direction ? { direction } : {}),
  label: `${title}, ${relation} of Pear rows`,
})

const NAMES: readonly RelatedName[] = [
  createName('Orchard plan', 'parent'),
  createName('Rootstock list', 'child'),
  createName('Grafting', 'jump'),
  createName('Pollination', 'reference', 'outgoing'),
  createName('Quince', 'reference', 'incoming'),
  createName('Bees', 'reference', 'both'),
  createName('Pruning', 'reference'),
]

const ICONS = [
  'arrow-up',
  'arrow-down',
  'arrow-left',
  'arrow-up-right',
  'arrow-down-left',
  'arrow-left-right',
  'arrow-left-right',
]

const wrappers: VueWrapper[] = []

const show = (props: { names: readonly RelatedName[]; showArrows?: boolean }) => {
  const wrapper = mount(RelatedNames, { props, attachTo: document.body })
  wrappers.push(wrapper)
  return wrapper
}

const getButtons = (wrapper: VueWrapper) => wrapper.findAll('button')

const getIconName = (button: Element | undefined): string | undefined =>
  Array.from(button?.querySelector('svg')?.classList ?? [])
    .find((one) => /^lucide-[a-z-]+$/.test(one) && !one.endsWith('-icon'))
    ?.replace(/^lucide-/, '')

afterEach(() => {
  while (wrappers.length > 0) wrappers.pop()?.unmount()
  document.body.replaceChildren()
})

describe('the names with the arrows on', () => {
  it('draws one arrow in each name, of the icon of its relation', async () => {
    const wrapper = show({ names: NAMES, showArrows: true })

    await vi.waitFor(
      () => expect(getButtons(wrapper).map((one) => getIconName(one.element))).toStrictEqual(ICONS),
      { timeout: ICON_TIMEOUT_MS },
    )
    expect(wrapper.findAll('svg')).toHaveLength(NAMES.length)
  })

  it('draws the arrow before the text of the name', async () => {
    const wrapper = show({ names: [createName('Grafting', 'jump')], showArrows: true })

    await vi.waitFor(() => expect(wrapper.find('button svg').exists()).toBe(true), {
      timeout: ICON_TIMEOUT_MS,
    })
    const button = wrapper.get('button').element

    expect(button.firstElementChild?.tagName.toLowerCase()).toBe('svg')
    expect(button.textContent?.trim()).toBe('Grafting')
  })

  it('hides the arrow from assistive technology and keeps the label of the name', async () => {
    const wrapper = show({ names: [createName('Grafting', 'jump')], showArrows: true })

    await vi.waitFor(() => expect(wrapper.find('button svg').exists()).toBe(true), {
      timeout: ICON_TIMEOUT_MS,
    })

    expect(wrapper.get('button svg').attributes('aria-hidden')).toBe('true')
    expect(wrapper.get('button').attributes('aria-label')).toBe('Grafting, jump of Pear rows')
  })

  it('chooses the path of a name that is clicked', async () => {
    const wrapper = show({ names: NAMES, showArrows: true })

    await getButtons(wrapper)[2]?.trigger('click')

    expect(wrapper.emitted('choose')).toStrictEqual([['notes/Grafting.md']])
  })
})

describe('the names with the arrows off', () => {
  it('draws no arrow', async () => {
    const wrapper = show({ names: NAMES, showArrows: false })
    await new Promise((resolve) => setTimeout(resolve, 200))

    expect(wrapper.findAll('svg')).toHaveLength(0)
    expect(getButtons(wrapper).map((one) => one.text())).toStrictEqual(
      NAMES.map((one) => one.title),
    )
  })

  it('keeps the label of each name', () => {
    const wrapper = show({ names: NAMES, showArrows: false })

    expect(getButtons(wrapper).map((one) => one.attributes('aria-label'))).toStrictEqual(
      NAMES.map((one) => one.label),
    )
  })

  it('chooses the path of a name that is clicked', async () => {
    const wrapper = show({ names: NAMES, showArrows: false })

    await getButtons(wrapper)[0]?.trigger('click')

    expect(wrapper.emitted('choose')).toStrictEqual([['notes/Orchard plan.md']])
  })
})
