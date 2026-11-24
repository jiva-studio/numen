import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import KeyFileUnlockDialog from './KeyFileUnlockDialog.vue'

describe('KeyFileUnlockDialog', () => {
  it('sends the passphrase and shows a refusal', async () => {
    const view = mount(KeyFileUnlockDialog, {
      props: { problem: 'This passphrase does not open your keys. Try again.', isBusy: false },
      attachTo: document.body,
    })
    await view.find('input[type="password"]').setValue('moon river')

    await view.find('form').trigger('submit')

    expect(view.emitted('unlock')).toEqual([['moon river']])
    expect(view.text()).toContain('does not open your keys')
    view.unmount()
  })

  it('sends nothing while the field is empty', async () => {
    const view = mount(KeyFileUnlockDialog, {
      props: { problem: '', isBusy: false },
      attachTo: document.body,
    })

    await view.find('form').trigger('submit')

    expect(view.emitted('unlock')).toBeUndefined()
    view.unmount()
  })
})
