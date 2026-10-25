/**
 * The permissions the plugin's manifest declares, which the window holds every call to.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const { permissions } = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as { permissions: readonly string[] }

describe('the manifest', () => {
  it('declares vault:write, which the notes the graph joins and makes need', () => {
    expect(permissions).toStrictEqual(['vault:read', 'vault:write'])
  })
})
