/** The icon that shows how a name is related to its note. */
import type { LinkDirection } from '@numen/plugin-sdk'
import type { NameRelation } from '../types'

/** The name of the icon, in the shared set, for a relation and, for a reference, the direction of its link. */
export function getRelationIcon(_relation: NameRelation, _direction?: LinkDirection): string {
  return ''
}
