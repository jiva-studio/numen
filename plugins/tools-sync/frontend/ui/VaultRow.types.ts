import type { VaultRowKind } from '../model/vaults'

export interface VaultRowProps {
  name: string
  kind: VaultRowKind
  status: string
  qualifier: string
  facts: readonly string[]
  isTransferring?: boolean
  progress?: number | null
}
