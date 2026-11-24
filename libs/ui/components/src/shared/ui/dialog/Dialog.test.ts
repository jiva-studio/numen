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
