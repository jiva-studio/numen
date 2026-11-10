/**
 * The title as the trigger of a list of vaults: what the button is, how it opens and closes, and
 * what a vault reports. The list is drawn at the end of the document, so it is read off the document.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import TitleBar from './TitleBar.vue'
import type { VaultsState, VaultSummary } from '@numen/plugin-sdk'

const ORCHARD: VaultSummary = { id: 'orchard', name: 'Orchard notes' }
const VAULTS: VaultsState = {
  vaults: [
    ORCHARD,
    { id: 'greenhouse', name: 'Greenhouse log' },
    { id: 'cellar', name: 'Cellar book' },
  ],
  current: ORCHARD,
}

const mounted: VueWrapper[] = []

const mountBar = (vaults: VaultsState | null = VAULTS): VueWrapper => {
  const wrapper = mount(TitleBar, {
    attachTo: document.body,
    props: {
      appName: 'numen',
      ...(vaults === null ? {} : { vaults }),
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

describe('the title with vaults', () => {
  it('is a button that says what it opens and whether it is open', () => {
    const trigger = getTrigger(mountBar())

    expect(trigger.element.tagName).toBe('BUTTON')
    expect(trigger.attributes('type')).toBe('button')
    expect(trigger.attributes('aria-haspopup')).toBe('dialog')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.text()).toBe('numen — Orchard notes')
  })

  it('is the plain title when the bar has no vaults', () => {
    for (const vaults of [null, { vaults: [], current: ORCHARD }, { ...VAULTS, current: null }]) {
      const wrapper = mountBar(vaults)
      expect(getTrigger(wrapper).element.tagName).toBe('SPAN')
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

  it('lists the vaults in the order given, the current one marked', async () => {
    await open(mountBar())

    expect(getLabels()).toStrictEqual(['Orchard notes', 'Greenhouse log', 'Cellar book'])
    expect(getOptions().map((one) => one.getAttribute('aria-current'))).toStrictEqual([
      'true',
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
  it('keeps the vaults that hold what is typed', async () => {
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

  it('lists only the current vault and nothing else when it is the only one', async () => {
    await open(mountBar({ vaults: [ORCHARD], current: ORCHARD }))

    expect(getLabels()).toStrictEqual(['Orchard notes'])
  })
})

describe('the row that creates a vault', () => {
  const mountWithCreate = (canCreateVault: boolean) => {
    const wrapper = mount(TitleBar, {
      attachTo: document.body,
      props: { appName: 'numen', vaults: VAULTS, canCreateVault },
    })
    mounted.push(wrapper)
    return wrapper
  }

  it('is the last row, below a line, while the window can create a vault', async () => {
    await open(mountWithCreate(true))

    expect(getLabels()).toStrictEqual([
      'Orchard notes',
      'Greenhouse log',
      'Cellar book',
      'Create vault',
    ])
    const separators = document.body.querySelectorAll('[role="separator"]')
    expect(separators).toHaveLength(1)
    expect(separators[0]?.nextElementSibling?.textContent?.trim()).toBe('Create vault')
  })

  it('is not drawn while the window cannot create a vault', async () => {
    await open(mountWithCreate(false))

    expect(getLabels()).toStrictEqual(['Orchard notes', 'Greenhouse log', 'Cellar book'])
    expect(document.body.querySelector('[role="separator"]')).toBeNull()
  })

  it('reports create-vault and no switch when it is chosen, and closes', async () => {
    const wrapper = mountWithCreate(true)
    await open(wrapper)

    getOptions().at(-1)!.click()
    await nextTick()
    await nextTick()

    expect(wrapper.emitted('create-vault')).toHaveLength(1)
    expect(wrapper.emitted('switch-vault')).toBeUndefined()
    expect(getFrame()).toBeNull()
  })
})

describe('choosing', () => {
  it('reports the id of a vault that is not current, and closes', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    getOptions()[1]!.click()
    await nextTick()
    await nextTick()

    expect(wrapper.emitted('switch-vault')).toStrictEqual([['greenhouse']])
    expect(getFrame()).toBeNull()
    expect(getTrigger(wrapper).attributes('aria-expanded')).toBe('false')
  })

  it('reports the vault the arrows and Enter reach', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    await press('ArrowDown')
    await press('ArrowDown')
    await press('Enter')

    expect(wrapper.emitted('switch-vault')).toStrictEqual([['cellar']])
  })

  it('reports nothing for the current vault and only closes', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    getOptions()[0]!.click()
    await nextTick()
    await nextTick()

    expect(wrapper.emitted('switch-vault')).toBeUndefined()
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
    expect(wrapper.emitted('switch-vault')).toBeUndefined()
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

  it('closes when the vaults go', async () => {
    const wrapper = mountBar()
    await open(wrapper)

    await wrapper.setProps({ vaults: { vaults: [], current: null } })
    await nextTick()

    expect(getFrame()).toBeNull()
    expect(getTrigger(wrapper).element.tagName).toBe('SPAN')
  })
})
