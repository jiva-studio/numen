/**
 * Menu item builders for the tab link context menu.
 */
import type { MenuItem } from '@numen/ui'
import type { EligibleLinkTab } from '../types'

export const UNLINK_ID = '__unlink__'
export const DETACH_ID = '__detach__'

export function createLinkMenuItems(
  eligibleTabs: readonly EligibleLinkTab[],
  isLinked: boolean = false,
  linkedTargetTitle: string = '',
  canDetach: boolean = false,
): MenuItem[] {
  const result: MenuItem[] = []

  if (canDetach) {
    result.push({
      id: DETACH_ID,
      text: 'Detach tab',
    })
  }

  if (isLinked) {
    result.push({
      id: UNLINK_ID,
      text: 'Unlink tab',
      ...(linkedTargetTitle ? { detail: `Linked with ${linkedTargetTitle}` } : {}),
    })
  }

  const groupName = 'Link with tab'

  for (const tab of eligibleTabs) {
    result.push({
      id: tab.id,
      text: tab.title || tab.kind,
      detail: tab.kind,
      group: groupName,
    })
  }

  return result
}
