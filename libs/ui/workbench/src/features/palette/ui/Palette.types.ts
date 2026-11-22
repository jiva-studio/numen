import type { Keystroke } from '@numen/ui'
import type { ActionWords } from '../lib/actions'
import type { PaletteGroup } from '../lib/item'
import type { PaletteMultiProps } from '../lib/multi'

export interface PaletteProps {
  groups?: readonly PaletteGroup[]
  isOpen?: boolean
  isBusy?: boolean
  placeholder?: string
  heading?: string | undefined
  crumb?: string
  step?: string
  opensOn?: string
  from?: HTMLElement | null
  to?: string | HTMLElement
  name?: string
  actionWords?: ActionWords
  actionKey?: Keystroke
  multi?: PaletteMultiProps | undefined
}
