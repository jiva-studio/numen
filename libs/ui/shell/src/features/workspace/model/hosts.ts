/**
 * The elements tab content is drawn into.
 *
 * A tab's content is rendered once by the workspace and placed into the host
 * element of the pane that currently holds the tab. Moving a tab to another pane
 * changes the host, and the content is moved there as it is.
 */
import { shallowReactive, type InjectionKey } from 'vue'
import type { NodeId, TabId } from '../lib/node'

interface Host {
  readonly pane: NodeId
  readonly element: HTMLElement
}

export interface TabHosts {
  /** The element a tab is drawn into, or undefined while no pane holds it. */
  readonly getElement: (tab: TabId) => HTMLElement | undefined
  /** Registers the element of a pane as the host of a tab. Repeating it changes nothing. */
  readonly attach: (tab: TabId, pane: NodeId, element: HTMLElement) => void
  /** Removes the host of a tab, unless another pane has taken it over since. */
  readonly detach: (tab: TabId, pane: NodeId) => void
}

export function createTabHosts(): TabHosts {
  const hosts = shallowReactive(new Map<TabId, Host>())
  return {
    getElement: (tab) => hosts.get(tab)?.element,
    attach: (tab, pane, element) => {
      const current = hosts.get(tab)
      if (current?.pane === pane && current.element === element) return
      hosts.set(tab, { pane, element })
    },
    detach: (tab, pane) => {
      if (hosts.get(tab)?.pane === pane) hosts.delete(tab)
    },
  }
}

export const TAB_HOSTS = Symbol('tab-hosts') as InjectionKey<TabHosts>
