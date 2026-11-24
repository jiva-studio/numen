import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useRecoveryProof } from '../../model/useRecoveryProof'
import IdentityRestoreDialog from './IdentityRestoreDialog.vue'

const getWords = (prefix: string): string[] =>
  Array.from({ length: 33 }, (_, index) => `${prefix}${String.fromCharCode(97 + (index % 26))}`)
const makeCode = (kit: string, sheet: number, prefix: string): string =>
  `numen://recover?v=1&kit=${kit}&sheet=${sheet}&words=${getWords(prefix).join('+')}`

// The camera is the outer edge: the stand-in only reports what a camera would scan.
const camera = defineComponent({
  name: 'QrScanner',
  emits: ['scanned', 'unavailable'],
  template: '<div class="camera" />',
})

function createDialog() {
  const Host = defineComponent({
    components: { IdentityRestoreDialog },
    setup: () => ({ proof: useRecoveryProof(ref('')) }),
    template: `<IdentityRestoreDialog :text="proof.text.value" :reading="proof.reading.value" problem="" :is-busy="false" @change="proof.setText" />`,
  })
  const wrapper = mount(Host, { attachTo: document.body, global: { stubs: { QrScanner: camera } } })
  const scan = async (text: string) => {
    wrapper.findComponent(camera).vm.$emit('scanned', text)
    await wrapper.vm.$nextTick()
  }
  const open = async () => {
    const button = wrapper.findAll('button').find((one) => one.text() === 'Scan QR')
    await button?.trigger('click')
  }
  const restore = () => wrapper.findAll('button').find((one) => one.text() === 'Restore')
  const lines = () => (wrapper.find('textarea').element as HTMLTextAreaElement).value.split('\n')
  return { wrapper, scan, open, restore, lines }
}

describe('the restore dialog scans the sheets of a kit', () => {
  it('shows 1 of 2 sheets after one scan and waits, then takes the second sheet', async () => {
    const { wrapper, scan, open, restore, lines } = createDialog()
    await open()
    await scan(makeCode('C0DE', 1, 'a'))
    expect(wrapper.text()).toContain('Sheet 1 of 2 scanned. Scan 1 more.')
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(restore()?.attributes('disabled')).toBeDefined()
    await scan(makeCode('C0DE', 3, 'b'))
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('.camera').exists()).toBe(false)
    expect(restore()?.attributes('disabled')).toBeUndefined()
    expect(lines()).toHaveLength(2)
  })

  it('refuses a sheet scanned twice and a sheet of another kit in plain words', async () => {
    const { wrapper, scan, open, lines } = createDialog()
    await open()
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 1, 'a'))
    expect(wrapper.find('[role=alert]').text()).toBe(
      'Sheet 1 is scanned already. Scan another sheet.',
    )
    await scan(makeCode('BEEF', 2, 'b'))
    expect(wrapper.find('[role=alert]').text()).toBe(
      'This sheet is from another kit. Scan a sheet of the same kit.',
    )
    expect(lines()).toHaveLength(1)
  })

  it('refuses a code that is no sheet', async () => {
    const { wrapper, scan, open } = createDialog()
    await open()
    await scan('https://example.com')
    expect(wrapper.find('[role=alert]').text()).toBe('This is not a code of a Numen recovery kit.')
  })
})

describe('the restore dialog has one mode at a time', () => {
  it('replaces the field with the camera, and Type words brings the field back', async () => {
    const { wrapper, open } = createDialog()
    expect(wrapper.find('textarea').exists()).toBe(true)
    await open()
    expect(wrapper.find('.camera').exists()).toBe(true)
    expect(wrapper.find('textarea').exists()).toBe(false)
    const back = wrapper.findAll('button').find((one) => one.text() === 'Type words')
    await back?.trigger('click')
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('.camera').exists()).toBe(false)
  })
})
