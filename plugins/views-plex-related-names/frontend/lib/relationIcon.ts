/** The icon that shows how a name is related to its note. */
import type { LinkDirection } from '@numen/plugin-sdk'
import type { NameRelation } from '../types'

const REFERENCE_ICONS: Readonly<Record<LinkDirection, string>> = {
  outgoing: 'arrow-up-right',
  incoming: 'arrow-down-left',
  both: 'arrow-left-right',
}

/** The name of the icon, in the shared set, for a relation and, for a reference, the direction of its link. */
export function getRelationIcon(relation: NameRelation, direction?: LinkDirection): string {
  switch (relation) {
    case 'parent':
      return 'arrow-up'
    case 'child':
      return 'arrow-down'
    case 'jump':
      return 'arrow-left'
    case 'reference':
      return REFERENCE_ICONS[direction ?? 'both']
  }
}
