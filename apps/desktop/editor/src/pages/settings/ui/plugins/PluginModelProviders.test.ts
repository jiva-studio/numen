import type { ProcessHost } from '@numen/plugin-loader'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { expect, it, vi } from 'vitest'
import PluginModelProviders from './PluginModelProviders.vue'

type ProcessUpdate = Parameters<ProcessHost['watch']>[0]

it('shows the stages of the selected running plugin until its model or process stops', async () => {
  let report: ProcessUpdate = () => undefined
  const dispose = vi.fn()
  const processes: ProcessHost = {
    watch: (handler) => {
      report = handler
      return { dispose }
    },
  }
  const model = {
    pluginId: 'org.example.recognition',
    pluginName: 'Recognition',
    startTime: 10,
    lastUseTime: 10,
    leaseCount: 1,
    isInProcess: false,
    health: 'ok' as const,
    healthText: '',
    startedBy: { who: 'Indexing', why: 'Read page' },
    holders: [],
    modelProviders: [
      { stage: 'Layout', provider: 'coreml' },
      { stage: 'Layout', provider: 'cuda' },
      { stage: 'Recognition', provider: 'cpu' },
    ],
  }
  const view = mount(PluginModelProviders, {
    props: {
      pluginId: model.pluginId,
      processes,
      options: [
        { value: 'coreml', title: 'Core ML' },
        { value: 'cpu', title: 'CPU' },
      ],
    },
  })
  expect(view.text()).toBe('')
  report([model], 10)
  await nextTick()
  expect(view.findAll('li').map((row) => row.text())).toEqual([
    'Layout · Core ML',
    'Layout · cuda',
    'Recognition · CPU',
  ])
  report([{ ...model, modelProviders: [] }], 11)
  await nextTick()
  expect(view.text()).toContain('No model loaded')
  report([{ ...model, pluginId: 'org.example.other' }], 12)
  await nextTick()
  expect(view.text()).toContain('Plugin is stopped')
  expect(view.findAll('li')).toHaveLength(0)
  view.unmount()
  expect(dispose).toHaveBeenCalledOnce()
})
