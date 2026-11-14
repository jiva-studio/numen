/**
 * The search part of the answers: the note names and passages a query finds, answered to the real
 * client over the router transport.
 */
import { mockVaultState } from '../answers'

export const searchAnswers = {
  search: {
    searchNames: () => ({ found: mockVaultState.names }),
    searchText: () => ({ found: mockVaultState.searchResults }),
    cancelSearch: () => ({}),
  },
}
