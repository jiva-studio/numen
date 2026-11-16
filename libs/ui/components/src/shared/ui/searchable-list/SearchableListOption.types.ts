import type { ListRow } from './rows'

export interface SearchableListOptionProps {
  row: ListRow
  id: string
  heading?: string | undefined
  hasSeparator?: boolean
  isActive: boolean
  isCurrent: boolean
  isMultiple?: boolean
}
