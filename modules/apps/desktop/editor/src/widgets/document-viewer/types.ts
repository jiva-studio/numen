/**
 * Types and interfaces for document viewing and navigation.
 */
import type { Span } from '../../shared/core'

/** Bounding rectangle on a page, in normalized fractions (0 to 1). */
export interface Rect {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** One page intrinsic size in points or pixels. */
export interface Page {
  readonly width: number
  readonly height: number
}

/** Highlights falling on one specific page. */
export interface PageHighlight {
  readonly page: number
  readonly rects: readonly Rect[]
}

/** Backward-compatible alias for PageHighlight. */
export type HighlightedPage = PageHighlight

/** Document layout manifest: page dimensions and disk fingerprint. */
export interface DocumentLayout {
  readonly pages: readonly Page[]
  readonly fingerprint: string
}

/** Backward-compatible alias for DocumentLayout. */
export interface Shape {
  readonly pages: readonly Page[]
  readonly at: string
}

/** Port for document service interactions. */
export interface Documents {
  /** Retrieves document layout and dimensions. */
  getDocumentLayout?(path: string): Promise<DocumentLayout>
  /** Retrieves document layout using legacy Shape contract. */
  getShape?(path: string): Promise<Shape | DocumentLayout>
  /** Returns the image URL for a rendered page at the given pixel width. */
  getPageUrl(path: string, page: number, width: number, seen?: string): string
  /** Retrieves highlights for given text spans. */
  getHighlights(
    path: string,
    spans: readonly Span[],
  ): Promise<readonly (readonly PageHighlight[])[]>
}

/** Port for measuring an attached page view. */
export interface PageHandle {
  measure(): void
}

