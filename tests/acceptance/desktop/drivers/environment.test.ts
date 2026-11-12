import { describe, expect, it } from 'vitest'
import { createRelease, getReleaseBuild, parseEnvironment } from './environment.js'

describe('parseEnvironment', () => {
  it('reads a name and a state from each row', () => {
    expect(parseEnvironment([['agent', 'slow'], ['latest release', '2.0.0'], ['embedding model', 'normal']])).toEqual([
      { name: 'agent', state: 'slow' },
      { name: 'latest release', state: '2.0.0' },
      { name: 'embedding model', state: 'normal' },
    ])
  })

  it('takes none as the state of the latest release', () => {
    expect(parseEnvironment([['latest release', 'none']])).toEqual([{ name: 'latest release', state: 'none' }])
  })

  it('fails on a name that is not part of the environment', () => {
    expect(() => parseEnvironment([['weather', 'slow']])).toThrow('is not part of the environment')
  })

  it('fails on a state the name does not take', () => {
    expect(() => parseEnvironment([['agent', 'none']])).toThrow('is not a state of the agent')
    expect(() => parseEnvironment([['latest release', 'soon']])).toThrow('is not a state of the latest release')
  })

  it('fails on an empty table and on a row that is not a pair', () => {
    expect(() => parseEnvironment([])).toThrow('is empty')
    expect(() => parseEnvironment([['agent']])).toThrow('must have a name and a state')
  })
})

describe('createRelease', () => {
  it('gives a later version a later build', () => {
    expect(getReleaseBuild('2.0.0')).toBeGreaterThan(getReleaseBuild('0.0.1'))
    expect(createRelease('2.0.0')).toMatchObject({ version: '2.0.0', build: '2000000' })
  })
})
