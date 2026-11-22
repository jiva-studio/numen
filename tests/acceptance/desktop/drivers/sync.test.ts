import { describe, expect, it } from 'vitest'
import { findBackendPids } from './sync.js'

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
