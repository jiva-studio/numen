/**
 * to the document and not only to one pane.
 */
import type { NodeId } from '../../lib/node'

export const getTabName = (pane: NodeId, pos: number): string => `${pane}-tab-${pos}`

export const getPanelName = (pane: NodeId, pos: number): string => `${pane}-panel-${pos}`
