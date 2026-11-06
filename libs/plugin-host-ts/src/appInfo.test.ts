import { describe, expect, it } from 'vitest'
import { createAppInfo, DEV_VERSION, getPlatform, parseBuildNumber } from './appInfo'

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
    expect(parseBuildNumber(text)).toBe(expected)
  })
})

describe('the platform', () => {
  it.each([
    ['Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0)', 'darwin'],
    ['Mozilla/5.0 (Windows NT 10.0; Win64; x64)', 'windows'],
    ['Mozilla/5.0 (X11; Linux x86_64)', 'linux'],
    ['', 'linux'],
  ])('%j is %s', (agent, expected) => {
    expect(getPlatform(agent)).toBe(expected)
  })
})

describe('the application info', () => {
  it('carries what the build was given', () => {
    expect(
      createAppInfo({ version: '2026.10.1', build: '884', userAgent: 'Windows NT 10.0' }),
    ).toEqual({ name: 'numen', version: '2026.10.1', build: 884, platform: 'windows' })
  })

  it('names a build made without a version the development build', () => {
    expect(createAppInfo({ userAgent: 'Linux' })).toEqual({
      name: 'numen',
      version: DEV_VERSION,
      build: 0,
      platform: 'linux',
    })
  })
})
