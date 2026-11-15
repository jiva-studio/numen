import { describe, expect, it } from 'vitest'
import { getEnvironment } from './environment.js'

describe('getEnvironment', () => {
  it('names the browser and the system', () => {
    expect(getEnvironment('webkit', 'darwin')).toMatchObject({
      id: 'webkit-darwin',
      label: 'WebKit · macOS',
    })
  })

  it('keeps an unknown project or system by its own name', () => {
    expect(getEnvironment('electron', 'freebsd').label).toBe('electron · freebsd')
  })

  it('orders by system, then by browser', () => {
    const order = (project: string, platform: string) => getEnvironment(project, platform).order
    expect(order('chromium', 'linux')).toBeLessThan(order('webkit', 'linux'))
    expect(order('webkit', 'linux')).toBeLessThan(order('chromium', 'darwin'))
  })
})
