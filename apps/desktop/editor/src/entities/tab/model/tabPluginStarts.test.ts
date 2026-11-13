import { describe, expect, it, vi } from 'vitest'
import { ReportedError } from '@/shared/api/errors'
import { createTabPluginStarts } from './tabPluginStarts'

const PLUGINS = [
  { id: 'org.example.scrolls', readers: [{ extensions: ['scroll', 'tome'] }] },
  { id: 'org.example.maps', readers: [{ extensions: ['atlas'] }] },
  { id: 'org.example.notes' },
]

const createStarts = () => {
  const started: string[] = []
  const writeMessage = vi.fn()
  const starts = createTabPluginStarts({
    status: {
      startPlugin: async (id) => {
        started.push(id)
      },
      startTabPlugins: async () => {
        throw new ReportedError('The chart kind is not declared')
      },
    },
    getPlugins: () => PLUGINS,
    writeMessage,
  })
  return { starts, started, writeMessage }
}

describe('the plugin starts of the tabs', () => {
  it('starts only the plugins whose readers open the extension', async () => {
    const { starts, started } = createStarts()

    await starts.startExtensionReader('tome')

    expect(started).toStrictEqual(['org.example.scrolls'])
  })

  it('reports a plugin of a tab kind that did not start', async () => {
    const { starts, writeMessage } = createStarts()

    starts.startKindPlugin('chart')
    await vi.waitFor(() => expect(writeMessage).toHaveBeenCalledOnce())

    expect(writeMessage).toHaveBeenCalledWith(
      'The plugin of the chart tab did not start: The chart kind is not declared',
      'error',
    )
  })
})
