import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
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
  props: ['continuous'],
  emits: ['scanned', 'unavailable'],
  template: '<div class="camera" />',
})

const mounted: VueWrapper[] = []

// The dialog is drawn into the body of the document, outside the wrapper.
const getBody = () => new DOMWrapper(document.body)

afterEach(() => {
  while (mounted.length) mounted.pop()?.unmount()
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
    template: `<RestoreSheetsDialog :slots="scanning.slots.value" :note="scanning.note.value" :problem="problem" :is-busy="isBusy" @scanned="scanning.scan" />`,
  })
  const wrapper = mount(Host, { attachTo: document.body, global: { stubs: { QrScanner: camera } } })
  mounted.push(wrapper)
  const scan = async (text: string) => {
    wrapper.findComponent(camera).vm.$emit('scanned', text)
    await flushPromises()
  }
  return { wrapper, scan, run, problem }
}

describe('the restore from sheets dialog', () => {
  it('has the camera and two waiting slots and no field', () => {
    createDialog()
    expect(getBody().find('.camera').exists()).toBe(true)
    expect(getBody().find('textarea').exists()).toBe(false)
    expect(getBody().find('input').exists()).toBe(false)
    expect(getBody().text()).toContain('First sheet')
    expect(getBody().text()).toContain('Second sheet')
    expect(getBody().text()).toContain('Take two of your three recovery sheets and scan their QR codes one after the other.')
  })

  it('keeps the camera on after the first sheet and restores by itself after the second', async () => {
    const { scan, run } = createDialog()
    await scan(makeCode('C0DE', 3, 'a'))
    expect(getBody().findAll('.restore-slots__sheet').map((one) => one.text())).toStrictEqual(['Sheet 3', ''])
    expect(getBody().text()).toContain('Second sheet')
    expect(getBody().find('.camera').exists()).toBe(true)
    expect(run).not.toHaveBeenCalled()
    await scan(makeCode('C0DE', 1, 'b'))
    expect(run).toHaveBeenCalledExactlyOnceWith({ sheets: [getWords('a'), getWords('b')] })
  })

  it('refuses a repeated sheet, another kit and a foreign code while it keeps scanning', async () => {
    const { scan, run } = createDialog()
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 1, 'a'))
    expect(getBody().get('[role=alert]').text()).toBe('Sheet 1 is scanned already. Scan another sheet.')
    await scan(makeCode('BEEF', 2, 'b'))
    expect(getBody().get('[role=alert]').text()).toBe(
      'This sheet is from another kit. Scan a sheet of the same kit.',
    )
    await scan('https://example.com')
    expect(getBody().get('[role=alert]').text()).toBe('This is not a code of a Numen recovery kit.')
    expect(run).not.toHaveBeenCalled()
    await scan(makeCode('C0DE', 2, 'c'))
    expect(run).toHaveBeenCalledOnce()
  })

  it('keeps one scanner in continuous mode through both sheets', async () => {
    const { wrapper, scan } = createDialog()
    const before = wrapper.findComponent(camera).vm
    expect(wrapper.findComponent(camera).props('continuous')).toBe(true)
    await scan(makeCode('C0DE', 3, 'a'))
    await scan(makeCode('C0DE', 1, 'b'))
    expect(wrapper.findComponent(camera).vm).toBe(before)
  })

  it('has only Cancel as a button', () => {
    createDialog()
    expect(getBody().findAll('button').map((one) => one.text())).toStrictEqual(['Cancel'])
  })

  it('shows the failure of the restore and starts over with the next sheet scanned', async () => {
    const { scan, problem } = createDialog(() => {
      problem.value = 'The sheets belong to another account.'
      return Promise.resolve()
    })
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 2, 'b'))
    expect(getBody().get('[role=alert]').text()).toBe('The sheets belong to another account.')
    expect(getBody().findAll('button').map((one) => one.text())).toStrictEqual(['Cancel'])
    problem.value = ''
    await scan(makeCode('C0DE', 3, 'c'))
    expect(getBody().find('[role=alert]').exists()).toBe(false)
    expect(getBody().findAll('.restore-slots__sheet').map((one) => one.text())).toStrictEqual(['Sheet 3', ''])
  })
})
