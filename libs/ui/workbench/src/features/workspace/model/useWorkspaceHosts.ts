import { provide } from 'vue'
import { createBrowserNodeIdFactory } from '../api/randomness'
import type { createNodeIdFactory } from '../lib/identity'
import { TAB_HOSTS, createTabHosts, type TabHosts } from './hosts'

/** Makes the elements tab content is drawn into and offers them to the panes below. */
export function useTabHosts(): TabHosts {
  const hosts = createTabHosts()
  provide(TAB_HOSTS, hosts)
  return hosts
}

/** The node identities the window's own randomness makes. */
export function useBrowserNodeIds(): ReturnType<typeof createNodeIdFactory> {
  return createBrowserNodeIdFactory()
}
