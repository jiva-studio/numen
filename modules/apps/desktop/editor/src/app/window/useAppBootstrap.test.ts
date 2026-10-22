/** What a window starts when it mounts and closes when it unmounts. */
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { useAppBootstrap, type BootstrapDeps } from './useAppBootstrap'

const setup = () => {
  const calls: string[] = []
  const note = (name: string) => () => {
    calls.push(name)
  }
  const noteLater = (name: string) => async () => {
    calls.push(name)
  }
  const deps: BootstrapDeps = {
    loadVaults: noteLater('loadVaults'),
    startSettings: noteLater('startSettings'),
    startWindow: noteLater('startWindow'),
    startEditing: note('startEditing'),
    startLayout: noteLater('startLayout'),
    startPlugins: noteLater('startPlugins'),
    closeWindow: note('closeWindow'),
    closeEditing: note('closeEditing'),
    closeTabs: note('closeTabs'),
    closeSettings: note('closeSettings'),
  }
  const view = mount(
    defineComponent({
      setup() {
        useAppBootstrap(deps)
        return () => h('div')
      },
    }),
  )
  return { calls, view }
}

describe('a window mounting', () => {
  it('loads the vaults, settings, plugins and layout, then starts the window and the editing', async () => {
    const { calls } = setup()

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(calls).toStrictEqual([
      'loadVaults',
      'startSettings',
      'startPlugins',
      'startLayout',
      'startWindow',
      'startEditing',
    ])
  })
})

describe('a window unmounting', () => {
  it('closes the window, the editing, the tabs and the settings, in that order', async () => {
    const { calls, view } = setup()
    await new Promise((resolve) => setTimeout(resolve, 0))
    calls.length = 0

    view.unmount()

    expect(calls).toStrictEqual(['closeWindow', 'closeEditing', 'closeTabs', 'closeSettings'])
  })
})
