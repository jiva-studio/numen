/** The parts of the window frame, and the CSS markers Wails reads from them. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import WindowControls from './WindowControls.vue'
import WindowDragArea from './WindowDragArea.vue'
import MacButtonsInset from './MacButtonsInset.vue'
import { MAC_BUTTONS_INSET, TITLE_BAR_HEIGHT } from '../lib/sizes'

const styleOf = (wrapper: VueWrapper, selector: string, name: string): string =>
  (wrapper.find(selector).element as HTMLElement).style.getPropertyValue(name).trim()

const CSS = ['windowFrame.css', '../tokens/frame.css']
  .map((file) => readFileSync(resolve(import.meta.dirname, file), 'utf8'))
  .join('\n')

/** The declarations of the first rule of the selector in the stylesheet. */
const readRule = (selector: string): string => {
  const start = CSS.indexOf(`${selector} {`)
  return start < 0 ? '' : CSS.slice(start, CSS.indexOf('}', start))
}

describe('WindowControls', () => {
  it('marks each button with the system button region of Windows', () => {
    const wrapper = mount(WindowControls, { props: { look: 'windows' } })

    expect(styleOf(wrapper, '[data-control="minimise"]', '--wails-non-client-region')).toBe(
      'minimize',
    )
    expect(styleOf(wrapper, '[data-control="toggle-maximise"]', '--wails-non-client-region')).toBe(
      'maximize',
    )
    expect(styleOf(wrapper, '[data-control="close"]', '--wails-non-client-region')).toBe('close')
  })

  it('keeps every button from moving the window', () => {
    const wrapper = mount(WindowControls, { props: { look: 'mac' } })

    for (const name of ['minimise', 'toggle-maximise', 'close']) {
      expect(styleOf(wrapper, `[data-control="${name}"]`, '--wails-draggable')).toBe('no-drag')
    }
  })

  it('emits the press of each button', async () => {
    const wrapper = mount(WindowControls, { props: { look: 'windows' } })

    await wrapper.find('[data-control="minimise"]').trigger('click')
    await wrapper.find('[data-control="toggle-maximise"]').trigger('click')
    await wrapper.find('[data-control="close"]').trigger('click')

    expect(wrapper.emitted('minimise')).toHaveLength(1)
    expect(wrapper.emitted('toggle-maximise')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('names the maximise button Restore while the window is maximised', () => {
    const wrapper = mount(WindowControls, { props: { look: 'windows', isMaximised: true } })

    expect(wrapper.find('[data-control="toggle-maximise"]').attributes('title')).toBe('Restore')
  })

  it('takes the names of the buttons it is given', () => {
    const wrapper = mount(WindowControls, {
      props: { look: 'windows', words: { close: 'Fermer' } },
    })

    expect(wrapper.find('[data-control="close"]').attributes('title')).toBe('Fermer')
    expect(wrapper.find('[data-control="minimise"]').attributes('title')).toBe('Minimise')
  })
})

describe('WindowDragArea', () => {
  it('moves the window by its contents', () => {
    const wrapper = mount(WindowDragArea, { slots: { default: '<span data-held>Orchard</span>' } })

    expect(styleOf(wrapper, '[data-window-drag]', '--wails-draggable')).toBe('drag')
    expect(wrapper.find('[data-held]').exists()).toBe(true)
  })
})

describe('MacButtonsInset', () => {
  it('keeps the room of the system buttons', () => {
    expect(mount(MacButtonsInset).find('[data-mac-buttons-inset]').exists()).toBe(true)
  })

  it('keeps no room in full screen, where macOS shows no buttons', () => {
    const wrapper = mount(MacButtonsInset, { props: { isFullscreen: true } })

    expect(wrapper.find('[data-mac-buttons-inset]').exists()).toBe(false)
  })
})

describe('the stylesheet of the frame', () => {
  it('lets Windows and Linux resize the frameless window by every edge', () => {
    expect(readRule('body')).toMatch(/--wails-resize:\s*all;/)
  })

  it('holds the sizes the frame reserves', () => {
    expect(readRule(':root')).toContain(`--numen-title-bar-height: ${TITLE_BAR_HEIGHT}px;`)
    expect(readRule(':root')).toContain(`--numen-mac-buttons-inset: ${MAC_BUTTONS_INSET}px;`)
  })
})
