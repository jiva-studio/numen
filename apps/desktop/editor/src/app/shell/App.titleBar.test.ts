/**
 * The title bar the window draws above its workspace.
 */
import { describe, expect, it } from 'vitest'
import { TitleBar } from '@numen/shell'
import { listed, mountWindow } from '@/testing/window'

describe('the title bar of the window', () => {
  it('is drawn above the workspace', async () => {
    const window = await mountWindow()

    const bar = window.find('[data-title-bar]')
    expect(bar.exists()).toBe(true)
    const order = window.find('main').element.children
    expect(order[0]?.hasAttribute('data-title-bar')).toBe(true)
  })

  it('names the application and the vault', async () => {
    const window = await mountWindow()

    expect(window.find('[data-title]').text()).toBe('numen — Physics')
  })

  it('names the application alone where no vault is open', async () => {
    listed.showing = ''

    const window = await mountWindow()

    expect(window.find('[data-title]').text()).toBe('numen')
  })

  it('draws the window controls and no plugin actions', async () => {
    const window = await mountWindow()

    const bar = window.findComponent(TitleBar)
    expect(bar.props('controls')).toBe('windows')
    expect(bar.props('actions')).toStrictEqual([])
    expect(window.findAll('[data-title-bar] button')).toHaveLength(3)
  })
})
