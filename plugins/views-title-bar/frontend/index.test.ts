import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick, type Component } from 'vue'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { TITLE_BAR_SLOT } from '@numen/plugin-sdk'
import { activate, TITLE_BAR_VIEW } from './index'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  id: string
  targets: readonly string[]
  contributes: { views: readonly { id: string; slot: string }[] }
  settings: readonly { key: string; default: string; options: readonly { value: string }[] }[]
}

const activateBar = (initialSettings: Record<string, unknown> = {}) => {
  const context = createMockPluginContext({ pluginId: manifest.id, initialSettings })
  activate(context)
  const [view] = context.getRegisteredSlotViews()
  return { context, view: view!.component as Component }
}

describe('the plugin', () => {
  it('registers the bar under the id the manifest declares in the title bar slot', () => {
    const { view } = activateBar()

    expect(view).toBeDefined()
    expect(manifest.contributes.views).toStrictEqual([{ id: TITLE_BAR_VIEW, slot: TITLE_BAR_SLOT }])
  })

  it('runs in the editor and in the flashcards window', () => {
    expect(manifest.id).toBe('md.numen.views.titleBar')
    expect(manifest.targets).toStrictEqual(['desktop.editor', 'desktop.flashcards'])
  })

  it('declares the setting of the window buttons, at the right by default', () => {
    const setting = manifest.settings.find((one) => one.key === 'buttons')

    expect(setting?.default).toBe('windows')
    expect(setting?.options.map((one) => one.value)).toStrictEqual(['windows', 'mac'])
  })
})

describe('the window buttons on Linux', () => {
  it('stand at the right while the setting says nothing', () => {
    const { view } = activateBar()

    const bar = mount(view, { props: { appName: 'numen', platform: 'linux' } })

    expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('windows')
  })

  it('stand at the left when the setting says mac, and move back when it changes', async () => {
    const { context, view } = activateBar({ buttons: 'mac' })
    const bar = mount(view, { props: { appName: 'numen', platform: 'linux' } })
    expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('mac')

    context.setSetting('buttons', 'windows')
    await nextTick()

    expect(bar.find('[data-title-bar]').attributes('data-controls')).toBe('windows')
  })

  it('stay with the system on macOS and at the right on Windows, whatever the setting', () => {
    const { view } = activateBar({ buttons: 'mac' })

    const mac = mount(view, { props: { appName: 'numen', platform: 'darwin' } })
    const windows = mount(view, { props: { appName: 'numen', platform: 'windows' } })

    expect(mac.find('[data-title-bar]').attributes('data-controls')).toBe('native-mac')
    expect(windows.find('[data-title-bar]').attributes('data-controls')).toBe('windows')
  })
})
