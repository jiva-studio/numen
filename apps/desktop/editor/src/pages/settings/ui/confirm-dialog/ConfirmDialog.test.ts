/**
 * What the dialog announces, where focus starts, and what answers it.
 */
import { mount } from '@vue/test-utils'
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

const getButton = (dialog: ReturnType<typeof mountDialog>, text: string) => {
  const found = dialog.findAll('button').find((one) => one.text() === text)
  if (!found) throw new Error(`no button ${text}`)
  return found
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ConfirmDialog', () => {
  it('is announced as a modal alert dialog with its label and question', () => {
    const dialog = mountDialog()
    const box = dialog.get('[role="alertdialog"]')
    expect(box.attributes('aria-modal')).toBe('true')
    expect(box.attributes('aria-label')).toBe('Delete page')
    expect(box.text()).toContain('Delete the page?')
  })

  it('points the dialog at its question as the description', () => {
    const dialog = mountDialog()
    const describedBy = dialog.get('[role="alertdialog"]').attributes('aria-describedby')
    expect(describedBy).toBeTruthy()
    expect(document.getElementById(describedBy!)?.textContent).toBe('Delete the page?')
  })

  it('starts focus on the cancel button', () => {
    mountDialog()
    expect(document.activeElement?.textContent?.trim()).toBe('Cancel')
  })

  it('emits confirm from the confirm button only', async () => {
    const dialog = mountDialog()
    await getButton(dialog, 'Delete').trigger('click')
    expect(dialog.emitted('confirm')).toHaveLength(1)
    expect(dialog.emitted('cancel')).toBeUndefined()
  })

  it('emits cancel from the cancel button, Escape and the backdrop', async () => {
    const dialog = mountDialog()
    await getButton(dialog, 'Cancel').trigger('click')
    await dialog.get('[role="alertdialog"]').trigger('keydown', { key: 'Escape' })
    await dialog.get('[data-slot="confirm-dialog-backdrop"]').trigger('click')
    expect(dialog.emitted('cancel')).toHaveLength(3)
    expect(dialog.emitted('confirm')).toBeUndefined()
  })

  it('does not cancel on a press inside the box', async () => {
    const dialog = mountDialog()
    await dialog.get('p').trigger('click')
    expect(dialog.emitted('cancel')).toBeUndefined()
  })

  it('marks the danger tone on the dialog and the default tone as default', () => {
    const danger = mountDialog({ tone: 'danger' })
    expect(danger.get('[role="alertdialog"]').attributes('data-tone')).toBe('danger')
    document.body.innerHTML = ''
    const neutral = mountDialog()
    expect(neutral.get('[role="alertdialog"]').attributes('data-tone')).toBe('default')
  })

  it('keeps Tab between the two buttons', async () => {
    const dialog = mountDialog()
    const cancel = getButton(dialog, 'Cancel')
    const confirm = getButton(dialog, 'Delete')
    ;(cancel.element as HTMLElement).focus()
    await cancel.trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(confirm.element)
    await confirm.trigger('keydown', { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(cancel.element)
  })
})
