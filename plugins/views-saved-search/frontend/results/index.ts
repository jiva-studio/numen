/**
 * The saved search tab: the kind of saved search notes, its component and its state.
 */
export { createSavedSearchKind, SAVED_SEARCH, type SavedSearchKindDeps } from './model/kind'
export type { SavedSearchTabState, SearchChange } from './model/tab'
export { createSavedSearchService, SERVICE } from './api/service'
export { default as SavedSearchTab } from './ui/SavedSearchTab.vue'
