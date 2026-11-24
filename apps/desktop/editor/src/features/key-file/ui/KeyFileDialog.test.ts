import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import KeyFileDialog from './KeyFileDialog.vue'

const open = (props: { problem?: string; isBusy?: boolean } = {}) =>
  mount(KeyFileDialog, {
    props: { problem: '', isBusy: false, ...props },
    attachTo: document.body,
  })

const getContinue = (view: ReturnType<typeof open>) =>
  view.findAll('button').find((one) => one.text().startsWith('Continue'))

describe('KeyFileDialog', () => {
  it('offers the passphrase first and waits for it to be repeated', async () => {
    const view = open()

    expect(document.body.textContent).toContain('Key storage')
    expect(document.body.textContent).toContain('Protect with passphrase (recommended)')
    expect(getContinue(view)?.attributes('disabled')).toBeDefined()
    view.unmount()
  })

  it('sends the passphrase once it matches its repeat', async () => {
    const view = open()
    const [first, second] = view.findAll('input[type="password"]')
    await first?.setValue('moon river')
    await second?.setValue('moon river')

    await view.find('form').trigger('submit')

    expect(view.emitted('choose')).toEqual([['passphrase', 'moon river']])
    view.unmount()
  })

  it('names a repeat that differs and sends nothing', async () => {
    const view = open()
    const [first, second] = view.findAll('input[type="password"]')
    await first?.setValue('moon river')
    await second?.setValue('moon')

    await view.find('form').trigger('submit')

    expect(view.text()).toContain('The two passphrases are not the same.')
    expect(view.emitted('choose')).toBeUndefined()
    view.unmount()
  })

  it('sends the plain file with the warning shown and no passphrase', async () => {
    const view = open()
    await view.findAll('input[type="radio"]')[1]?.setValue()

    expect(view.text()).toContain('Anyone who can read your files can read your keys.')
    await view.find('form').trigger('submit')

    expect(view.emitted('choose')).toEqual([['plaintext', '']])
    view.unmount()
  })

  it('cancels', async () => {
    const view = open()

    await view.findAll('button').find((one) => one.text() === 'Cancel')?.trigger('click')

    expect(view.emitted('cancel')).toHaveLength(1)
    view.unmount()
  })
})
