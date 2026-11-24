/** The shared modal: its roles, what closes it, and where the focus goes. */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import ModalDialog from './ModalDialog.vue'

const mountDialog = (props: Record<string, unknown> = {}) =>
  mount(ModalDialog, {
    attachTo: document.body,
    props: { label: 'Restore identity', hint: 'Enter the words.', ...props },
    slots: {
      default: '<input aria-label="Words" />',
      actions: '<button type="button">Cancel</button><button type="submit">Go</button>',
    },
  })

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ModalDialog', () => {
  it('names the dialog by its title and draws the hint', () => {
    const dialog = mountDialog()
    const box = dialog.get('[role="dialog"]')
    expect(box.attributes('aria-modal')).toBe('true')
    expect(document.getElementById(box.attributes('aria-labelledby')!)?.textContent).toBe(
      'Restore identity',
    )
    expect(dialog.text()).toContain('Enter the words.')
  })

  it('focuses the first control when it opens', () => {
    mountDialog()
    expect(document.activeElement?.getAttribute('aria-label')).toBe('Words')
  })

  it('emits cancel on Escape and on the backdrop, not on a press inside', async () => {
    const dialog = mountDialog()
    await dialog.get('[role="dialog"]').trigger('click')
    expect(dialog.emitted('cancel')).toBeUndefined()
    await dialog.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    await dialog.get('[data-slot="modal-dialog-backdrop"]').trigger('click')
    expect(dialog.emitted('cancel')).toHaveLength(2)
  })

  it('is a form that emits submit when asked', async () => {
    const dialog = mountDialog({ isForm: true })
    await dialog.get('form').trigger('submit')
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
