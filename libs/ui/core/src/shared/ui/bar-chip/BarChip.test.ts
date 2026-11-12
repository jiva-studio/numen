/**
 * What the bar chip names, emits and draws, and what it does not. Where its popover is placed is
 * read in `BarChip.stories.ts`, in a browser.
 *
 * The popover is drawn at the end of the document, so it is read off the document and not off the
 * wrapper.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, h, ref } from 'vue'
import BarChip from './BarChip.vue'
import { FACE, LABEL, POPOVER_ACTION, POPOVER_LABEL, POPOVER_TEXT, TOOLTIP } from './fixtures/faces'

const mounted: VueWrapper[] = []

afterEach(() => {
  for (const wrapper of mounted.splice(0)) wrapper.unmount()
})

const drawChip = (
  props: Record<string, unknown> = {},
  slots: Record<string, () => unknown> = { popover: () => h('p', POPOVER_TEXT) },
) => {
  const wrapper = mount(BarChip, {
    props: { label: LABEL, tooltip: TOOLTIP, popoverLabel: POPOVER_LABEL, ...props },
    slots: { default: () => FACE, ...slots },
    attachTo: document.body,
  })
  mounted.push(wrapper)
  return wrapper
}

const getTrigger = (): HTMLButtonElement => {
  const trigger = document.body.querySelector<HTMLButtonElement>(`button[aria-label="${LABEL}"]`)
  if (!trigger) throw new Error('the chip draws no trigger')
  return trigger
}

const getDialog = (): HTMLElement | null =>
  document.body.querySelector<HTMLElement>('[role="dialog"]')

const press = async (element: HTMLElement): Promise<void> => {
  element.click()
  await flushPromises()
}

const pressEscape = async (): Promise<void> => {
  document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
  await flushPromises()
}

describe('the trigger of the bar chip', () => {
  it('is a button named by the label, with the tooltip as its title, out of the tab order', () => {
    drawChip()
    const trigger = getTrigger()

    expect(trigger.getAttribute('type')).toBe('button')
    expect(trigger.getAttribute('title')).toBe(TOOLTIP)
    expect(trigger.getAttribute('aria-haspopup')).toBe('dialog')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    expect(trigger.getAttribute('tabindex')).toBe('-1')
    expect(trigger.textContent?.trim()).toBe(FACE)
  })

  it('has no title when no tooltip is given', () => {
    drawChip({ tooltip: undefined })

    expect(getTrigger().hasAttribute('title')).toBe(false)
  })
})

describe('the popover of the bar chip', () => {
  it('opens on a press as a dialog named by the popover label and emits open', async () => {
    const wrapper = drawChip()
    expect(getDialog()).toBeNull()

    await press(getTrigger())

    const dialog = getDialog()
    expect(dialog?.getAttribute('aria-label')).toBe(POPOVER_LABEL)
    expect(dialog?.textContent).toContain(POPOVER_TEXT)
    expect(getTrigger().getAttribute('aria-expanded')).toBe('true')
    expect(wrapper.emitted('update:open')).toStrictEqual([[true]])
  })

  it('closes on Escape and emits that it is closed', async () => {
    const wrapper = drawChip()
    await press(getTrigger())

    await pressEscape()

    expect(getDialog()).toBeNull()
    expect(getTrigger().getAttribute('aria-expanded')).toBe('false')
    expect(wrapper.emitted('update:open')).toStrictEqual([[true], [false]])
  })

  it('opens nothing and emits no open when it has no popover content', async () => {
    const wrapper = drawChip({}, {})

    await press(getTrigger())

    expect(getDialog()).toBeNull()
    expect(getTrigger().getAttribute('aria-expanded')).toBe('false')
    expect(wrapper.emitted('update:open') ?? []).not.toContainEqual([true])
  })

  it('keeps the popover open when the window is resized', async () => {
    drawChip()
    await press(getTrigger())

    window.dispatchEvent(new Event('resize'))
    await flushPromises()

    expect(getDialog()).not.toBeNull()
  })

  it('closes when the parent sets open to false from an action in the content', async () => {
    const Host = defineComponent({
      setup() {
        const isOpen = ref(false)
        return () =>
          h(
            BarChip,
            {
              label: LABEL,
              popoverLabel: POPOVER_LABEL,
              open: isOpen.value,
              'onUpdate:open': (value: boolean) => (isOpen.value = value),
            },
            {
              default: () => FACE,
              popover: () =>
                h(
                  'button',
                  { type: 'button', onClick: () => (isOpen.value = false) },
                  POPOVER_ACTION,
                ),
            },
          )
      },
    })
    mounted.push(mount(Host, { attachTo: document.body }))
    await press(getTrigger())
    const action = [...(getDialog()?.querySelectorAll('button') ?? [])].find(
      (button) => button.textContent === POPOVER_ACTION,
    )
    expect(action).toBeDefined()

    await press(action!)

    expect(getDialog()).toBeNull()
    expect(getTrigger().getAttribute('aria-expanded')).toBe('false')
  })

  it('opens from the parent when open is set to true', async () => {
    const wrapper = drawChip({ open: false })

    await wrapper.setProps({ open: true })
    await flushPromises()

    expect(getDialog()).not.toBeNull()
  })
})
