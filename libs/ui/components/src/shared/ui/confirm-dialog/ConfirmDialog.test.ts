/**
 * What the dialog announces, where focus starts, and what answers it.
 */
import { DOMWrapper, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import ConfirmDialog from './ConfirmDialog.vue'

type DialogProps = InstanceType<typeof ConfirmDialog>['$props']

const mountDialog = (props: Partial<DialogProps> = {}) =>
  mount(ConfirmDialog, {
    attachTo: document.body,
    props: {
      text: 'Delete the page?',
      label: 'Delete page',
      confirmText: 'Delete',
      cancelText: 'Cancel',
      ...props,
    },
  })

const getBody = () => new DOMWrapper(document.body)

const getButton = (text: string) => {
  const found = getBody().findAll('button').find((one) => one.text() === text)
  if (!found) throw new Error(`no button ${text}`)
  return found
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ConfirmDialog', () => {
  it('is announced as a modal alert dialog named by its title, with its question', () => {
    mountDialog()
    const box = getBody().get('[role="alertdialog"]')
    expect(box.attributes('aria-modal')).toBe('true')
    const labelledBy = box.attributes('aria-labelledby')
    expect(labelledBy).toBeTruthy()
    expect(document.getElementById(labelledBy!)?.textContent).toBe('Delete page')
    expect(box.text()).toContain('Delete the page?')
  })

  it('points the dialog at its question as the description', () => {
    mountDialog()
    const describedBy = getBody().get('[role="alertdialog"]').attributes('aria-describedby')
    expect(describedBy).toBeTruthy()
    expect(document.getElementById(describedBy!)?.textContent).toBe('Delete the page?')
  })

  it('draws what the default slot holds in place of the text, as the description', () => {
    mount(ConfirmDialog, {
      attachTo: document.body,
      props: { label: 'Switch to Pro?', confirmText: 'Switch to Pro', cancelText: 'Cancel' },
      slots: { default: '<dl><dt>New plan</dt><dd>Pro, 250 GB, $30 / month</dd></dl>' },
    })
    const describedBy = getBody().get('[role="alertdialog"]').attributes('aria-describedby')
    expect(document.getElementById(describedBy!)?.querySelector('dd')?.textContent).toBe(
      'Pro, 250 GB, $30 / month',
    )
  })

  it('starts focus on the cancel button', () => {
    mountDialog()
    expect(document.activeElement?.textContent?.trim()).toBe('Cancel')
  })

  it('emits confirm from the confirm button only', async () => {
    const dialog = mountDialog()
    await getButton('Delete').trigger('click')
    expect(dialog.emitted('confirm')).toHaveLength(1)
    expect(dialog.emitted('cancel')).toBeUndefined()
  })

  it('emits cancel from the cancel button, Escape and the backdrop', async () => {
    const dialog = mountDialog()
    await getButton('Cancel').trigger('click')
    await getBody().get('[role="alertdialog"]').trigger('keydown', { key: 'Escape' })
    await getBody().get('[data-slot="dialog-backdrop"]').trigger('click')
    expect(dialog.emitted('cancel')).toHaveLength(3)
    expect(dialog.emitted('confirm')).toBeUndefined()
  })

  it('does not cancel on a press inside the box', async () => {
    const dialog = mountDialog()
    await getBody().get('.confirm-dialog__text').trigger('click')
    expect(dialog.emitted('cancel')).toBeUndefined()
  })

  it('marks the danger tone on the dialog and the default tone as default', () => {
    mountDialog({ tone: 'danger' })
    expect(getBody().get('[role="alertdialog"]').attributes('data-tone')).toBe('danger')
    document.body.innerHTML = ''
    mountDialog()
    expect(getBody().get('[role="alertdialog"]').attributes('data-tone')).toBe('default')
  })

  it('wraps Tab from the last button to the first and back', async () => {
    mountDialog()
    const cancel = getButton('Cancel')
    const confirm = getButton('Delete')
    ;(confirm.element as HTMLElement).focus()
    await confirm.trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(cancel.element)
    await cancel.trigger('keydown', { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(confirm.element)
  })
})
