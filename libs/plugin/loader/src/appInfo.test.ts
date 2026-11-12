import { describe, expect, it } from 'vitest'
import { createAppInfo, DEV_VERSION } from './appInfo'
import type { AppInfo } from '@numen/plugin-sdk'

describe('the build number', () => {
  it.each([
    ['884', 884],
    [' 12 ', 12],
    ['0', 0],
    ['-3', 0],
    ['abc', 0],
    ['', 0],
    [undefined, 0],
  ])('%j is %i', (text, expected) => {
    expect(createAppInfo({ build: text, getPlatform: () => 'linux' }).build).toBe(expected)
  })
})

describe('the application info', () => {
  it('carries what the build was given', () => {
    expect(
      createAppInfo({ version: '2026.10.1', build: '884', getPlatform: () => 'windows' }),
    ).toEqual({ name: 'numen', version: '2026.10.1', build: 884, platform: 'windows' })
  })

  it('names a build made without a version the development build', () => {
    expect(createAppInfo({ getPlatform: () => 'linux' })).toEqual({
      name: 'numen',
      version: DEV_VERSION,
      build: 0,
      platform: 'linux',
    })
  })

  it('reports the platform the window reports at the time it is read', () => {
    let platform: AppInfo['platform'] = 'windows'
    const app = createAppInfo({ getPlatform: () => platform })

    expect(app.platform).toBe('windows')
    platform = 'darwin'
    expect(app.platform).toBe('darwin')
  })
})
