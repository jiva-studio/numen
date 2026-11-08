import { describe, expect, it } from 'vitest'
import { getWindowRadius, readMajorVersion } from './corners'

describe('readMajorVersion', () => {
  it.each([
    ['26.6.2', 26],
    ['26.0', 26],
    ['15.6.1', 15],
    ['14', 14],
  ])('reads %s as %i', (version, major) => {
    expect(readMajorVersion(version)).toBe(major)
  })

  it.each([undefined, '', 'Tahoe'])('reads %j as unknown', (version) => {
    expect(readMajorVersion(version)).toBeUndefined()
  })
})

describe('getWindowRadius', () => {
  it.each([
    ['darwin', 26, false, 26],
    ['darwin', 27, false, 26],
    ['darwin', 15, false, 10],
    ['darwin', 11, false, 10],
    ['darwin', undefined, false, 26],
    ['darwin', 26, true, 0],
    ['darwin', 15, true, 0],
    ['windows', undefined, false, 0],
    ['windows', 26, false, 0],
    ['linux', undefined, false, 0],
  ] as const)('%s, version %j, full screen %j: %i', (platform, major, isFullscreen, radius) => {
    expect(getWindowRadius(platform, major, isFullscreen)).toBe(radius)
  })
})
