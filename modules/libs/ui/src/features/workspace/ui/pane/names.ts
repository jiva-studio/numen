/**
 * A tab and the panel it stands over name each other, so the names are unique
 * to the document and not only to one pane.
 */
import type { NodeId } from '../../lib/node'

export const getTabName = (pane: NodeId, at: number): string => `${pane}-tab-${at}`

export const getPanelName = (pane: NodeId, at: number): string => `${pane}-panel-${at}`
