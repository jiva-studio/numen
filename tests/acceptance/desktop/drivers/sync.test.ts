import { describe, expect, it } from 'vitest'
import { findBackendPids, findChildBackendPids, getIntervalSeconds, parsePlans } from './sync.js'

const LISTING = [
  '    1 /sbin/init',
  ' 4242 /home/p/.local/share/numen/plugins/md.numen.tools.sync/bin/tools-sync',
  ' 4243 /home/p/.local/share/numen/plugins/md.numen.tools.sync/bin/tools-sync --flag',
  ' 4300 /home/p/plugins/tools-sync/bin/syncaccount seed',
  ' 4400 grep tools-sync',
].join('\n')

describe('findBackendPids', () => {
  it('finds the processes whose program is the backend file of the plugin', () => {
    expect(findBackendPids(LISTING, 'tools-sync')).toStrictEqual([4242, 4243])
  })

  it('finds none where no process runs the file', () => {
    expect(findBackendPids(LISTING, 'tools-other')).toStrictEqual([])
    expect(findBackendPids('', 'tools-sync')).toStrictEqual([])
  })
})

describe('findChildBackendPids', () => {
  it('finds only the sync backend started by the selected computer', () => {
    const processes = [
      ' 105 102 /tmp/first/plugins/tools-sync/bin/tools-sync --flag',
      ' 102 101 /tmp/first/numen-debug',
      ' 110 107 /tmp/second/plugins/tools-sync/bin/tools-sync',
      ' 107 1 /tmp/second/numen-debug',
      ' 109 101 /tmp/first/plugins/tools-logs/bin/tools-logs',
      ' 101 1 /tmp/first/numen-debug',
    ].join('\n')

    expect(findChildBackendPids(processes, 101, 'tools-sync')).toEqual([105])
    expect(findChildBackendPids(processes, 107, 'tools-sync')).toEqual([110])
    expect(findChildBackendPids(processes, 200, 'tools-sync')).toEqual([])
  })
})

describe('parsePlans', () => {
  it('reads the plans of a list joined by commas and and', () => {
    expect(parsePlans('"Basic" with 10 GB, "Plus" with 100 GB and "Pro" with 250 GB')).toStrictEqual([
      { name: 'Basic', gigabytes: 10 },
      { name: 'Plus', gigabytes: 100 },
      { name: 'Pro', gigabytes: 250 },
    ])
  })

  it('reads one plan', () => {
    expect(parsePlans('"Basic" with 10 GB')).toStrictEqual([{ name: 'Basic', gigabytes: 10 }])
  })
})

describe('getIntervalSeconds', () => {
  it('reads the units a scenario names', () => {
    expect(getIntervalSeconds('1 second')).toBe(1)
    expect(getIntervalSeconds('1 hour')).toBe(3600)
  })

  it('refuses a text that is not a number of a unit', () => {
    expect(() => getIntervalSeconds('often')).toThrow('does not name an interval')
  })
})
