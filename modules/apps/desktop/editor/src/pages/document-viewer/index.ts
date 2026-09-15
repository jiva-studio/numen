/** The document tab: the pages it shows and the vault it reads them from. */
export { default as DocumentTab } from './ui/DocumentTab.vue'
export { createDocumentKind } from './kind'
export { documents } from './api/wire'
export { useDocumentReader } from './model/useDocumentReader'
export { useDocumentTab } from './model/useDocumentTab'
export type { Documents } from './types'
