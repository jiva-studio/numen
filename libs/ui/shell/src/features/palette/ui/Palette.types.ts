import type { PaletteKeys } from '@numen/ui'
import type { ActionWords } from '../lib/actions'
import type { PaletteGroup } from '../lib/item'

export interface PaletteProps {
  groups?: readonly PaletteGroup[]
  isOpen?: boolean
  /** More results are on their way; the field shows a spinner. */
  isBusy?: boolean
  placeholder?: string
  crumb?: string
  step?: string
  opensOn?: string
  from?: HTMLElement | null
  to?: string | HTMLElement
  name?: string
  actionWords?: ActionWords
  actionKey?: PaletteKeys
}
