import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { dispose, getScreen } from '@/testing/dom'
import KeyFileDialog from './KeyFileDialog.vue'

let view: VueWrapper | undefined
afterEach(() => {
  if (view !== undefined) dispose(view)
  view = undefined
})

const open = (): VueWrapper => {
  view = mount(KeyFileDialog, {
    props: { problem: '', isBusy: false },
    attachTo: document.body,
  })
  return view
}

const getContinue = () =>
  getScreen()
    .findAll('button')
    .find((one) => one.text().startsWith('Continue'))

describe('KeyFileDialog', () => {
  it('offers the passphrase first and waits for it to be repeated', () => {
    open()

    expect(document.body.textContent).toContain('Key storage')
    expect(document.body.textContent).toContain('Protect with passphrase (recommended)')
    expect(getContinue()?.attributes('disabled')).toBeDefined()
  })

  it('sends the passphrase once it matches its repeat', async () => {
    const opened = open()
    const [first, second] = getScreen().findAll('input[type="password"]')
    await first?.setValue('moon river')
    await second?.setValue('moon river')

    await getScreen().find('form').trigger('submit')

    expect(opened.emitted('choose')).toEqual([['passphrase', 'moon river']])
  })

  it('names a repeat that differs and sends nothing', async () => {
    const opened = open()
    const [first, second] = getScreen().findAll('input[type="password"]')
    await first?.setValue('moon river')
    await second?.setValue('moon')

    await getScreen().find('form').trigger('submit')

    expect(getScreen().text()).toContain('The two passphrases are not the same.')
    expect(opened.emitted('choose')).toBeUndefined()
  })

  it('sends the plain file with the warning shown and no passphrase', async () => {
    const opened = open()
    await getScreen().findAll('input[type="radio"]')[1]?.setValue()

    expect(getScreen().text()).toContain('Anyone with access to your files can read the key.')
    await getScreen().find('form').trigger('submit')

    expect(opened.emitted('choose')).toEqual([['plaintext', '']])
  })

  it('cancels', async () => {
    const opened = open()

    await getScreen()
      .findAll('button')
      .find((one) => one.text() === 'Cancel')
      ?.trigger('click')

    expect(opened.emitted('cancel')).toHaveLength(1)
  })
})
