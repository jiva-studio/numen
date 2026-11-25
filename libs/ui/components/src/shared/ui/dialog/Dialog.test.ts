/** The shared modal: its roles, what closes it, and where the focus goes. */
import { DOMWrapper, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import Dialog from './Dialog.vue'
import { provideDialogHost } from './host'

const mountDialog = (props: Record<string, unknown> = {}) =>
  mount(Dialog, {
    attachTo: document.body,
    props: { label: 'Restore identity', hint: 'Enter the words.', ...props },
    slots: {
      default: '<input aria-label="Words" />',
      actions: '<button type="button">Cancel</button><button type="submit">Go</button>',
    },
  })

const getBody = () => new DOMWrapper(document.body)

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Dialog', () => {
  it('names the dialog by its title and draws the hint', () => {
    mountDialog()
    const box = getBody().get('[role="dialog"]')
    expect(box.attributes('aria-modal')).toBe('true')
    expect(document.getElementById(box.attributes('aria-labelledby')!)?.textContent).toBe(
      'Restore identity',
    )
    expect(getBody().text()).toContain('Enter the words.')
  })

  it('focuses the first control when it opens', () => {
    mountDialog()
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Words')
  })

  it('emits cancel on Escape and on the backdrop, not on a press inside', async () => {
    const dialog = mountDialog()
    await getBody().get('[role="dialog"]').trigger('click')
    expect(dialog.emitted('cancel')).toBeUndefined()
    await getBody().get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    await getBody().get('[data-slot="dialog-backdrop"]').trigger('click')
    expect(dialog.emitted('cancel')).toHaveLength(2)
  })

  it('emits submit from a form dialog', async () => {
    const dialog = mountDialog({ isForm: true })
    await getBody().get('form').trigger('submit')
    expect(dialog.emitted('submit')).toHaveLength(1)
  })

  it('returns the focus to the control that had it before', () => {
    const opener = document.createElement('button')
    document.body.append(opener)
    opener.focus()
    const dialog = mountDialog()
    dialog.unmount()
    expect(document.activeElement).toBe(opener)
  })
})

describe('Dialog accessibility', () => {
  it('describes the dialog by its hint', () => {
    mountDialog()
    const box = getBody().get('[role="dialog"]')
    expect(document.getElementById(box.attributes('aria-describedby')!)?.textContent).toBe(
      'Enter the words.',
    )
  })

  it('describes the dialog by the element given, not by the hint', () => {
    mountDialog({ describedBy: 'elsewhere' })
    expect(getBody().get('[role="dialog"]').attributes('aria-describedby')).toBe('elsewhere')
  })

  it('has no description when it has no hint', () => {
    mountDialog({ hint: '' })
    expect(getBody().get('[role="dialog"]').attributes('aria-describedby')).toBeUndefined()
  })

  it('makes the rest of the page inert while it is open and restores it after', () => {
    const page = document.createElement('main')
    const kept = document.createElement('aside')
    kept.inert = true
    document.body.append(page, kept)
    const dialog = mountDialog()
    expect(page.inert).toBe(true)
    expect(getBody().get('[data-slot="dialog-backdrop"]').element.inert).toBeFalsy()
    dialog.unmount()
    expect(page.inert).toBe(false)
    expect(kept.inert).toBe(true)
  })

  it('focuses the panel when it has no control', () => {
    const dialog = mount(Dialog, { attachTo: document.body, props: { label: 'Done' } })
    expect(document.activeElement).toBe(getBody().get('[role="dialog"]').element)
    dialog.unmount()
  })

  it('keeps the focus on the panel when Tab is pressed and it has no control', async () => {
    mount(Dialog, { attachTo: document.body, props: { label: 'Done' } })
    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    getBody().get('[role="dialog"]').element.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(getBody().get('[role="dialog"]').element)
  })

  it('skips a control that tabindex takes out of the order and wraps at a control that tabindex adds', async () => {
    mount(Dialog, {
      attachTo: document.body,
      props: { label: 'Pick' },
      slots: {
        default:
          '<button>First</button><div tabindex="0" data-testid="last">Last</div><button tabindex="-1">Skipped</button>',
      },
    })
    const last = getBody().get('[data-testid="last"]').element as HTMLElement
    last.focus()
    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    last.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    expect(document.activeElement?.textContent).toBe('First')
  })
})

describe('Dialog host', () => {
  const mountDialogInHost = () => {
    const host = document.createElement('main')
    document.body.append(host)
    const Root = defineComponent({
      setup() {
        provideDialogHost(ref(host))
        return () => h(Dialog, { label: 'Restore identity' })
      },
    })
    mount(Root, { attachTo: document.body })
    return { host }
  }

  it('draws into the host element when one is provided', () => {
    const { host } = mountDialogInHost()
    expect(host.querySelector('[role="dialog"]')).not.toBeNull()
    expect(document.body.querySelector(':scope > [data-slot="dialog-backdrop"]')).toBeNull()
  })

  it('draws into the body when no host is provided', () => {
    mountDialog()
    expect(document.body.querySelector(':scope > [data-slot="dialog-backdrop"]')).not.toBeNull()
  })
})
