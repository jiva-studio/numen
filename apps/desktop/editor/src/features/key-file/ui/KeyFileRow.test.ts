import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { KeyFileState } from '@/entities/key-file'
import KeyFileRow from './KeyFileRow.vue'

const state = (overrides: Partial<KeyFileState> = {}): KeyFileState => ({
  isInUse: true,
  mode: 'passphrase',
  isChoiceNeeded: false,
  isLocked: false,
  warning: '',
  ...overrides,
})

describe('KeyFileRow', () => {
  it('offers to change how the keys are kept', async () => {
    const view = mount(KeyFileRow, { props: { state: state() } })

    await view.find('button').trigger('click')

    expect(view.text()).toContain('In a file protected by your passphrase.')
    expect(view.emitted('change')).toHaveLength(1)
  })

  it('offers to enter the passphrase while the file is locked', async () => {
    const view = mount(KeyFileRow, { props: { state: state({ isLocked: true }) } })

    await view.find('button').trigger('click')

    expect(view.emitted('unlock')).toHaveLength(1)
  })

  it('says a plain file can be read by anyone and has no action on the keychain', () => {
    const plain = mount(KeyFileRow, { props: { state: state({ mode: 'plaintext' }) } })
    const keychain = mount(KeyFileRow, { props: { state: state({ isInUse: false }) } })

    expect(plain.text()).toContain('Anyone who can read your files')
    expect(keychain.find('button').exists()).toBe(false)
  })
})
