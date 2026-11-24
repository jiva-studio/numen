/** The form that reads the QR code of the other device: the camera where there is one, the text field always. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { click, dispose, enterText, findButton, textInputs } from '@/testing/dom'
import LinkScanForm from './LinkScanForm.vue'

let wrapper: VueWrapper | undefined
afterEach(() => {
  if (wrapper !== undefined) dispose(wrapper)
  wrapper = undefined
})

const mountForm = async (
  findCamera: () => Promise<boolean>,
  camera: () => Promise<MediaStream> = () => new Promise<MediaStream>(() => undefined),
) => {
  wrapper = mount(LinkScanForm, {
    props: { isBusy: false, findCamera, camera },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

describe('the scan form', () => {
  it('shows the camera and the text field where the machine has a camera', async () => {
    const form = await mountForm(() => Promise.resolve(true))
    expect(form.find('video').exists()).toBe(true)
    expect(textInputs()).toHaveLength(1)
    expect(form.text()).not.toContain('No camera found')
  })

  it('asks for the pasted code where the machine has no camera', async () => {
    const form = await mountForm(() => Promise.resolve(false))
    expect(form.find('video').exists()).toBe(false)
    expect(form.get('[role="status"]').text()).toBe(
      'No camera found: paste the code from the other device.',
    )
    expect(textInputs()).toHaveLength(1)
  })

  it('switches to the text field when the camera is not allowed', async () => {
    const form = await mountForm(
      () => Promise.resolve(true),
      () => Promise.reject(new DOMException('no', 'NotAllowedError')),
    )
    expect(form.find('video').exists()).toBe(false)
    expect(form.get('[role="status"]').text()).toBe(
      'The camera is not allowed. Paste the code from the other device.',
    )
  })

  it('switches to the text field when the camera did not open', async () => {
    const form = await mountForm(
      () => Promise.resolve(true),
      () => Promise.reject(new DOMException('busy', 'NotReadableError')),
    )
    expect(form.get('[role="status"]').text()).toBe(
      'The camera is used by another program. Paste the code from the other device.',
    )
  })

  it('submits the pasted code where the machine has no camera', async () => {
    const form = await mountForm(() => Promise.resolve(false))
    await enterText(textInputs()[0] as HTMLInputElement, 'numen://link?v=1')
    await click(findButton(/^Use link$/))
    expect(form.emitted('submit')).toStrictEqual([['numen://link?v=1']])
  })

  it('waits for the check before it shows the camera', async () => {
    const check = vi.fn(() => new Promise<boolean>(() => undefined))
    const form = await mountForm(check)
    expect(form.find('video').exists()).toBe(false)
    expect(form.find('[role="status"]').exists()).toBe(false)
  })
})
