import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { expect, it } from 'vitest'
import { readRemovalTarget } from './syncRemoval.js'

it('finds the specific synced file id and its linked vault before removal', () => {
  const store = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-removal-record-'))
  const vaultId = '1'.repeat(32)
  const target = Buffer.alloc(16, 2)
  try {
    fs.writeFileSync(path.join(store, 'link.json'), JSON.stringify({ remoteVaultId: vaultId }))
    fs.writeFileSync(path.join(store, 'files.json'), JSON.stringify({
      files: [
        { path: 'Plan.md', fileId: Buffer.alloc(16, 3).toString('base64') },
        { path: 'Shopping list.md', fileId: target.toString('base64') },
      ],
    }))

    expect(readRemovalTarget(store, 'Shopping list.md')).toEqual({ vaultId, fileId: target.toString('hex') })
    expect(() => readRemovalTarget(store, 'Other.md')).toThrow('the file has no synced record')
  } finally {
    fs.rmSync(store, { recursive: true })
  }
})
