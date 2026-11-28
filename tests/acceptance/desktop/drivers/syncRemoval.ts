import fs from 'node:fs'
import path from 'node:path'

export interface RemovalTarget {
  readonly vaultId: string
  readonly fileId: string
}

export const isRemoteId = (value: unknown): value is string =>
  typeof value === 'string' && /^[0-9a-f]{32}$/.test(value)

export function readRemovalTarget(store: string, filePath: string): RemovalTarget {
  const link = JSON.parse(fs.readFileSync(path.join(store, 'link.json'), 'utf8')) as { remoteVaultId?: unknown }
  const records = JSON.parse(fs.readFileSync(path.join(store, 'files.json'), 'utf8')) as {
    files?: readonly { readonly path: string; readonly fileId: string }[]
  }
  const encoded = records.files?.find((record) => record.path === filePath)?.fileId
  if (!isRemoteId(link.remoteVaultId) || encoded === undefined) {
    throw new Error('the file has no synced record at ' + filePath)
  }
  const fileId = Buffer.from(encoded, 'base64')
  if (fileId.length !== 16 || fileId.toString('base64') !== encoded) {
    throw new Error('the synced file record has an invalid id for ' + filePath)
  }
  return { vaultId: link.remoteVaultId, fileId: fileId.toString('hex') }
}
