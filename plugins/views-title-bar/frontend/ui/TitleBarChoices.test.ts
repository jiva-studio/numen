/**
 * The title as the trigger of a list of choices: what the button is, how it opens and closes, and
 * what a choice reports. The list is drawn at the end of the document, so it is read off the document.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import TitleBar from './TitleBar.vue'
import type { TitleBarChoice } from '@numen/plugin-sdk'

const CHOICES: readonly TitleBarChoice[] = [
  { id: 'orchard', label: 'Orchard notes', isCurrent: true },
  { id: 'greenhouse', label: 'Greenhouse log' },
  { id: 'cellar', label: 'Cellar book' },
  { id: 'add', label: 'Add a folder' },
]

const mounted: VueWrapper[] = []

const mountBar = (choices: readonly TitleBarChoice[] | null = CHOICES): VueWrapper => {
  const wrapper = mount(TitleBar, {
    attachTo: document.body,
    props: {
      appName: 'numen',
      vaultName: 'Orchard notes',
      ...(choices === null ? {} : { choices }),
    },
  })
  mounted.push(wrapper)
  return wrapper
}

const getTrigger = (wrapper: VueWrapper) => wrapper.find('[data-title]')
const getFrame = () => document.body.querySelector<HTMLElement>('[data-numen-searchable-list]')
const getInput = () =>
  document.body.querySelector<HTMLInputElement>('[data-numen-searchable-list] input')
const getOptions = () => [...document.body.querySelectorAll<HTMLElement>('[role="option"]')]
const getLabels = () => getOptions().map((one) => one.textContent?.trim())

const open = async (wrapper: VueWrapper): Promise<void> => {
  await getTrigger(wrapper).trigger('click')
  await nextTick()
  await nextTick()
}

const type = async (text: string): Promise<void> => {
  const input = getInput()!
  input.value = text
  input.dispatchEvent(new Event('input', { bubbles: true }))
  await nextTick()
}

const press = async (key: string): Promise<void> => {
  getInput()!.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
  await nextTick()
  await nextTick()
}

afterEach(() => {
  mounted.splice(0).forEach((one) => one.unmount())
  document.body.innerHTML = ''
})

describe('the title with choices', () => {
  it('is a button that says what it opens and whether it is open', () => {
    const trigger = getTrigger(mountBar())

    expect(trigger.element.tagName).toBe('BUTTON')
    expect(trigger.attributes('type')).toBe('button')
    expect(trigger.attributes('aria-haspopup')).toBe('dialog')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.text()).toBe('numen — Orchard notes')
  })

  it('stays out of the drag area while the rest of the bar drags', () => {
    const wrapper = mountBar()
    const drag = (element: Element) =>
      (element as HTMLElement).style.getPropertyValue('--wails-draggable')

    expect(drag(getTrigger(wrapper).element)).toBe('no-drag')
    expect(drag(wrapper.find('[data-title-bar]').element)).toBe('drag')
    expect(wrapper.find('[data-title-fill]').exists()).toBe(true)
    expect(drag(wrapper.find('[data-title-fill]').element)).toBe('')
  })

  it('is the plain title when the bar has no choices', () => {
    for (const choices of [null, []] as const) {
      const wrapper = mountBar(choices)
      expect(getTrigger(wrapper).element.tagName).toBe('SPAN')
      expect(wrapper.find('[data-title-fill]').exists()).toBe(false)
    }
  })
})

describe('opening the list', () => {
  it('opens on a press of the title and puts the keyboard on the search field', async () => {
    const wrapper = mountBar()

    await open(wrapper)

    expect(getTrigger(wrapper).attributes('aria-expanded')).toBe('true')
    expect(getFrame()).not.toBeNull()
    expect(document.activeElement).toBe(getInput())
  })

  it('opens from the keyboard, as a button does', async () => {
    const wrapper = mountBar()
    const trigger = getTrigger(wrapper).element as HTMLButtonElement
    trigger.focus()

    trigger.click()
    await nextTick()
    await nextTick()

    expect(getFrame()).not.toBeNull()
  })

  it('lists the choices in the order given, the current one marked', async () => {
    await open(mountBar())

    expect(getLabels()).toStrictEqual([
      'Orchard notes',
      'Greenhouse log',
      'Cellar book',
      'Add a folder',
    ])
    expect(getOptions().map((one) => one.getAttribute('aria-current'))).toStrictEqual([
      'true',
      null,
      null,
      null,
    ])
  })

  it('names the list and the search field as the vault switcher', async () => {
    await open(mountBar())

    expect(getFrame()?.getAttribute('aria-label')).toBe('Switch vault')
    expect(getInput()?.getAttribute('placeholder')).toBe('Search vaults')
  })
})

describe('searching the list', () => {
  it('keeps the choices that hold what is typed', async () => {
    await open(mountBar())

    await type('green')

    expect(getLabels()).toStrictEqual(['Greenhouse log'])
  })

  it('says in one line that nothing was found', async () => {
    await open(mountBar())

    await type('zzz')

    expect(getOptions()).toHaveLength(0)
    expect(getFrame()?.querySelector('[role="status"]')?.textContent?.trim()).toBe('No vault found')
  })

  it('lists only the current choice and nothing else when it is the only one', async () => {
    await open(mountBar([CHOICES[0]!]))

    expect(getLabels()).toStrictEqual(['Orchard notes'])
  })
})

describe('choosing', () => {
  it('reports the id of a choice that is not current, and closes', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    getOptions()[1]!.click()
    await nextTick()
    await nextTick()

    expect(wrapper.emitted('choose')).toStrictEqual([['greenhouse']])
    expect(getFrame()).toBeNull()
    expect(getTrigger(wrapper).attributes('aria-expanded')).toBe('false')
  })

  it('reports the choice the arrows and Enter reach', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    await press('ArrowDown')
    await press('ArrowDown')
    await press('Enter')

    expect(wrapper.emitted('choose')).toStrictEqual([['cellar']])
  })

  it('reports nothing for the current choice and only closes', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    getOptions()[0]!.click()
    await nextTick()
    await nextTick()

    expect(wrapper.emitted('choose')).toBeUndefined()
    expect(getFrame()).toBeNull()
  })
})

describe('closing the list', () => {
  it('closes on Escape and gives the keyboard back to the title', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    await press('Escape')

    expect(getFrame()).toBeNull()
    expect(document.activeElement).toBe(getTrigger(wrapper).element)
    expect(wrapper.emitted('choose')).toBeUndefined()
  })

  it('closes on a press outside it', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    document.body.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await nextTick()
    await nextTick()

    expect(getFrame()).toBeNull()
  })

  it('closes on a press of the title and does not open again', async () => {
    const wrapper = mountBar()
    await open(wrapper)
    const trigger = getTrigger(wrapper).element

    trigger.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    await nextTick()
    await getTrigger(wrapper).trigger('click')
    await nextTick()
    await nextTick()

    expect(getFrame()).toBeNull()
    expect(getTrigger(wrapper).attributes('aria-expanded')).toBe('false')
  })

  it('closes when the choices go', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    await wrapper.setProps({ choices: [] })
    await nextTick()

    expect(getFrame()).toBeNull()
    expect(getTrigger(wrapper).element.tagName).toBe('SPAN')
  })
})
