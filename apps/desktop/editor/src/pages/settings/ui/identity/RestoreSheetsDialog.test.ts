import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { KitProof } from '@/entities/identity'
import { useSheetScan } from '../../model/useSheetScan'
import RestoreSheetsDialog from './RestoreSheetsDialog.vue'

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

function createDialog(restore: (proof: KitProof) => Promise<void> = () => Promise.resolve()) {
  const problem = ref('')
  const isBusy = ref(false)
  const run = vi.fn(async (proof: KitProof) => {
    isBusy.value = true
    await restore(proof)
    isBusy.value = false
  })
  const Host = defineComponent({
    components: { RestoreSheetsDialog },
    setup: () => ({ scanning: useSheetScan(problem, run), problem, isBusy }),
    template: `<RestoreSheetsDialog :slots="scanning.slots.value" :note="scanning.note.value" :problem="problem" :is-busy="isBusy" @scanned="scanning.scan" @retry="scanning.retry" />`,
  })
  const wrapper = mount(Host, { attachTo: document.body, global: { stubs: { QrScanner: camera } } })
  const scan = async (text: string) => {
    wrapper.findComponent(camera).vm.$emit('scanned', text)
    await flushPromises()
  }
  return { wrapper, scan, run, problem }
}

describe('the restore from sheets dialog', () => {
  it('has the camera and two waiting slots and no field', () => {
    const { wrapper } = createDialog()
    expect(wrapper.find('.camera').exists()).toBe(true)
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(wrapper.find('input').exists()).toBe(false)
    expect(wrapper.text()).toContain('Sheet 1 waiting')
    expect(wrapper.text()).toContain('Sheet 2 waiting')
    expect(wrapper.text()).toContain('Scan any 2 of your 3 sheets.')
  })

  it('keeps the camera on after the first sheet and restores by itself after the second', async () => {
    const { wrapper, scan, run } = createDialog()
    await scan(makeCode('C0DE', 3, 'a'))
    expect(wrapper.text()).toContain('Sheet 3 scanned')
    expect(wrapper.text()).toContain('Sheet 2 waiting')
    expect(wrapper.find('.camera').exists()).toBe(true)
    expect(run).not.toHaveBeenCalled()
    await scan(makeCode('C0DE', 1, 'b'))
    expect(run).toHaveBeenCalledExactlyOnceWith({ sheets: [getWords('a'), getWords('b')] })
  })

  it('refuses a repeated sheet, another kit and a foreign code while it keeps scanning', async () => {
    const { wrapper, scan, run } = createDialog()
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 1, 'a'))
    expect(wrapper.get('[role=alert]').text()).toBe('Sheet 1 is scanned already. Scan another sheet.')
    await scan(makeCode('BEEF', 2, 'b'))
    expect(wrapper.get('[role=alert]').text()).toBe(
      'This sheet is from another kit. Scan a sheet of the same kit.',
    )
    await scan('https://example.com')
    expect(wrapper.get('[role=alert]').text()).toBe('This is not a code of a Numen recovery kit.')
    expect(run).not.toHaveBeenCalled()
    await scan(makeCode('C0DE', 2, 'c'))
    expect(run).toHaveBeenCalledOnce()
  })

  it('shows the failure of the restore with Try again, which empties the slots', async () => {
    const { wrapper, scan, problem } = createDialog(() => {
      problem.value = 'The sheets belong to another account.'
      return Promise.resolve()
    })
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 2, 'b'))
    expect(wrapper.get('[role=alert]').text()).toBe('The sheets belong to another account.')
    const retry = wrapper.findAll('button').find((one) => one.text() === 'Try again')
    await retry?.trigger('click')
    expect(wrapper.text()).toContain('Sheet 1 waiting')
    expect(wrapper.find('[role=alert]').exists()).toBe(false)
  })
})
