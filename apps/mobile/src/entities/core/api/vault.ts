import type { Transport } from '@connectrpc/connect'
import { createVaultClient } from '@numen/core-client'
import type { Core } from '../lib/core'

/** The calls of the vault service over `transport`, with every answer in the window's own types. */
export function createVaultApi(transport: Transport): Pick<Core, 'getVaultState'> {
  const vault = createVaultClient(transport)
  return {
    async getVaultState() {
      const answer = await vault.getVaultState({})
      return { id: answer.id, name: answer.name, path: answer.path }
    },
  }
}
