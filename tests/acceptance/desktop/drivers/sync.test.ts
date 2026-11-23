import { describe, expect, it } from 'vitest'
import { findBackendPids, getCloudBytes, getPlansFlag, getSubscriptionPhase, parsePlans } from './sync.js'

const LISTING = [
  '    1 /sbin/init',
  ' 4242 /home/p/.local/share/numen/plugins/md.numen.tools.sync/bin/tools-sync',
  ' 4243 /home/p/.local/share/numen/plugins/md.numen.tools.sync/bin/tools-sync --flag',
  ' 4300 /home/p/plugins/tools-sync/bin/fakesync -addr 127.0.0.1:0',
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

describe('getSubscriptionPhase', () => {
  it('is grace for a subscription that ended fewer than 7 days ago', () => {
    expect(getSubscriptionPhase(3)).toBe('grace')
    expect(getSubscriptionPhase(6)).toBe('grace')
  })

  it('is read only from 7 days to fewer than 60 days after the end', () => {
    expect(getSubscriptionPhase(7)).toBe('read_only')
    expect(getSubscriptionPhase(10)).toBe('read_only')
    expect(getSubscriptionPhase(59)).toBe('read_only')
  })

  it('is deleted from 60 days after the end', () => {
    expect(getSubscriptionPhase(60)).toBe('deleted')
    expect(getSubscriptionPhase(70)).toBe('deleted')
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

describe('getPlansFlag', () => {
  it('names each plan by its name in lower case and its size in GB', () => {
    const plans = [
      { name: 'Basic', gigabytes: 10 },
      { name: 'Plus', gigabytes: 100 },
      { name: 'Pro', gigabytes: 250 },
    ]

    expect(getPlansFlag(plans)).toBe('basic:10,plus:100,pro:250')
  })
})

describe('getCloudBytes', () => {
  it('counts a GB as 2 to the power of 30 bytes', () => {
    expect(getCloudBytes(2)).toBe(2 * 1024 * 1024 * 1024)
    expect(getCloudBytes(0)).toBe(0)
  })
})
