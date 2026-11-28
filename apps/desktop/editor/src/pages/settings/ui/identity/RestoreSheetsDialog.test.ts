import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { KitProof } from '@/entities/identity'
import { createFakeCamera } from '@/testing/fakeCamera'
import { useSheetScan } from '../../model/useSheetScan'
import RestoreSheetsDialog from './RestoreSheetsDialog.vue'

const getWords = (prefix: string): string[] =>
  Array.from({ length: 33 }, (_, index) => `${prefix}${String.fromCharCode(97 + (index % 26))}`)
const makeCode = (kit: string, sheet: number, prefix: string): string =>
  `numen://recover?v=1&kit=${kit}&sheet=${sheet}&words=${getWords(prefix).join('+')}`

const mounted: VueWrapper[] = []

const getBody = () => new DOMWrapper(document.body)

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  while (mounted.length) mounted.pop()?.unmount()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

async function createDialog(restore: (proof: KitProof) => Promise<void> = () => Promise.resolve()) {
  const problem = ref('')
  const isBusy = ref(false)
  const run = vi.fn(async (proof: KitProof) => {
    isBusy.value = true
    await restore(proof)
    isBusy.value = false
  })
  const eye = createFakeCamera()
  const Host = defineComponent({
    components: { RestoreSheetsDialog },
    setup: () => ({ scanning: useSheetScan(problem, run), problem, isBusy, camera: eye.camera }),
    template: `<RestoreSheetsDialog :slots="scanning.slots.value" :note="scanning.note.value" :problem="problem" :is-busy="isBusy" :camera="camera" @scanned="scanning.scan" />`,
  })
  const wrapper = mount(Host, { attachTo: document.body })
  mounted.push(wrapper)
  await flushPromises()
  let last = ''
  const scan = async (text: string) => {
    if (text === last) {
      eye.hide()
      await eye.waitFrames(17)
    }
    eye.show(text)
    await eye.waitFrames(1)
    last = text
    await flushPromises()
  }
  return { wrapper, scan, run, problem, eye }
}

describe('the restore from sheets dialog', () => {
  it('has the camera and two waiting slots and no field', async () => {
    await createDialog()
    expect(getBody().find('video').exists()).toBe(true)
    expect(getBody().find('textarea').exists()).toBe(false)
    expect(getBody().find('input').exists()).toBe(false)
    expect(getBody().text()).toContain('First sheet')
    expect(getBody().text()).toContain('Second sheet')
    expect(getBody().text()).toContain(
      'Take two of your three recovery sheets and scan their QR codes one after the other.',
    )
  })

  it('keeps the camera on after the first sheet and restores by itself after the second', async () => {
    const { scan, run } = await createDialog()
    await scan(makeCode('C0DE', 3, 'a'))
    expect(
      getBody()
        .findAll('.stage-list__detail')
        .map((one) => one.text()),
    ).toStrictEqual(['Sheet 3', '', ''])
    expect(getBody().text()).toContain('Second sheet')
    expect(getBody().find('video').exists()).toBe(true)
    expect(run).not.toHaveBeenCalled()
    await scan(makeCode('C0DE', 1, 'b'))
    expect(run).toHaveBeenCalledExactlyOnceWith({ sheets: [getWords('a'), getWords('b')] })
  })

  it('refuses a repeated sheet, another kit and a foreign code while it keeps scanning', async () => {
    const { scan, run } = await createDialog()
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 1, 'a'))
    expect(getBody().get('[role=alert]').text()).toBe(
      'Sheet 1 is scanned already. Scan another sheet.',
    )
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
    const { scan, eye } = await createDialog()
    await scan(makeCode('C0DE', 3, 'a'))
    await scan(makeCode('C0DE', 1, 'b'))
    expect(eye.camera).toHaveBeenCalledOnce()
    expect(eye.stop).not.toHaveBeenCalled()
  })

  it('lists the three stages in order before any scan', async () => {
    await createDialog()
    const rows = getBody().findAll('.stage-list__row')
    expect(rows.map((row) => row.get('.stage-list__label').text())).toStrictEqual([
      'First sheet',
      'Second sheet',
      'Restore',
    ])
    expect(rows.map((row) => row.attributes('data-state'))).toStrictEqual([
      'active',
      'pending',
      'pending',
    ])
  })

  it('keeps a scanned sheet in its row as done and shows the restore as active while it runs', async () => {
    let finish = (): void => undefined
    const { scan } = await createDialog(() => new Promise<void>((resolve) => (finish = resolve)))
    await scan(makeCode('C0DE', 3, 'a'))
    expect(
      getBody()
        .findAll('.stage-list__row')
        .map((row) => row.attributes('data-state')),
    ).toStrictEqual(['done', 'active', 'pending'])
    await scan(makeCode('C0DE', 1, 'b'))
    const rows = getBody().findAll('.stage-list__row')
    expect(rows.map((row) => row.attributes('data-state'))).toStrictEqual([
      'done',
      'done',
      'active',
    ])
    expect(rows[2]?.text()).toContain('Restoring…')
    expect(rows.map((row) => row.get('.stage-list__label').text())).toStrictEqual([
      'First sheet',
      'Second sheet',
      'Restore',
    ])
    finish()
  })

  it('keeps the list after a failure and sets the sheets back to pending', async () => {
    const { scan, problem } = await createDialog(() => {
      problem.value = 'The sheets belong to another identity.'
      return Promise.resolve()
    })
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 2, 'b'))
    const rows = getBody().findAll('.stage-list__row')
    expect(rows).toHaveLength(3)
    expect(rows.map((row) => row.attributes('data-state'))).toStrictEqual([
      'active',
      'pending',
      'pending',
    ])
  })

  it('has only Cancel as a button', async () => {
    await createDialog()
    expect(
      getBody()
        .findAll('button')
        .map((one) => one.text()),
    ).toStrictEqual(['Cancel'])
  })

  it('shows the failure of the restore and starts over with the next sheet scanned', async () => {
    const { scan, problem } = await createDialog(() => {
      problem.value = 'The sheets belong to another identity.'
      return Promise.resolve()
    })
    await scan(makeCode('C0DE', 1, 'a'))
    await scan(makeCode('C0DE', 2, 'b'))
    expect(getBody().get('[role=alert]').text()).toBe('The sheets belong to another identity.')
    expect(
      getBody()
        .findAll('button')
        .map((one) => one.text()),
    ).toStrictEqual(['Cancel'])
    await scan(makeCode('C0DE', 3, 'c'))
    expect(getBody().find('[role=alert]').exists()).toBe(false)
    expect(
      getBody()
        .findAll('.stage-list__detail')
        .map((one) => one.text()),
    ).toStrictEqual(['Sheet 3', '', ''])
  })
})
