import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { dispose, getScreen } from '@/testing/dom'
import KeyFileUnlockDialog from './KeyFileUnlockDialog.vue'

let view: VueWrapper | undefined
afterEach(() => {
  if (view !== undefined) dispose(view)
  view = undefined
})

const open = (problem: string): VueWrapper => {
  view = mount(KeyFileUnlockDialog, {
    props: { problem, isBusy: false },
    attachTo: document.body,
  })
  return view
}

describe('KeyFileUnlockDialog', () => {
  it('sends the passphrase and shows a refusal', async () => {
    const opened = open('This passphrase does not open your keys. Try again.')
    await getScreen().find('input[type="password"]').setValue('moon river')

    await getScreen().find('form').trigger('submit')

    expect(opened.emitted('unlock')).toEqual([['moon river']])
    expect(getScreen().text()).toContain('does not open your keys')
  })

  it('sends nothing while the field is empty', async () => {
    const opened = open('')

    await getScreen().find('form').trigger('submit')

    expect(opened.emitted('unlock')).toBeUndefined()
  })
})
