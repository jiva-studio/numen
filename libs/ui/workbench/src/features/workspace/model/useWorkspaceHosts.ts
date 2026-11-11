import { provide } from 'vue'
import { createBrowserNodeIdFactory } from '../api/randomness'
import { TAB_HOSTS, createTabHosts } from './hosts'

/**
 * The elements tab content is drawn into, offered to the panes below, and the node identities the
 * window's own randomness makes.
 */
export function useWorkspaceHosts() {
  const hosts = createTabHosts()
  provide(TAB_HOSTS, hosts)
  return { hosts, createNodeId: createBrowserNodeIdFactory() }
}
