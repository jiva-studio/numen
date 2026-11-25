import { DOMWrapper, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import RestoreTypedDialog from './RestoreTypedDialog.vue'
import RestoreWordsDialog from './RestoreWordsDialog.vue'

const mounted: VueWrapper[] = []

const getBody = () => new DOMWrapper(document.body)

afterEach(() => {
  while (mounted.length) mounted.pop()?.unmount()
})

const WARNING =
  'The sync server cannot be checked against your recovery kit. If you have another device, confirm on it that the devices listed are yours.'

describe('the restore dialogs', () => {
  it('tell the person that the sync server cannot be checked against the recovery kit', () => {
    mounted.push(
      mount(RestoreWordsDialog, {
        attachTo: document.body,
        props: { text: '', reading: { proof: undefined, note: '', isProblem: false }, problem: '', isBusy: false },
      }),
    )

    expect(getBody().text()).toContain(WARNING)
  })

  it('tell it before a restore from typed sheets too', () => {
    mounted.push(
      mount(RestoreTypedDialog, {
        attachTo: document.body,
        props: {
          sheets: ['', ''],
          reading: { proof: undefined, note: '', isProblem: false },
          problem: '',
          isBusy: false,
        },
      }),
    )

    expect(getBody().text()).toContain(WARNING)
  })
})
