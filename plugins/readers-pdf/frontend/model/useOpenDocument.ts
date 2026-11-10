/**
 * The tab of one PDF: the document read from the files the window serves, or from the documents
 * the caller passes.
 */
import { documents as servedDocuments } from '../api/wire'
import type { Documents } from '../types'
import { useDocumentReader } from './useDocumentReader'
import { useDocumentTab } from './useDocumentTab'

export function useOpenDocument(path: string, documents: Documents = servedDocuments) {
  return useDocumentTab(useDocumentReader(documents, path))
}
