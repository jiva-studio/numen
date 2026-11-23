/**
 * The search part of the answers: the note names and passages a query finds, answered to the real
 * client over the router transport.
 */
import { mockVaultState } from '../answers'

export const searchAnswers = {
  search: {
    searchNames: () => ({ matches: mockVaultState.names }),
    searchText: () => ({ results: mockVaultState.searchResults }),
    cancelSearch: () => ({}),
  },
}
