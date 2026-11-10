/**
 * Types and interfaces for PDF document viewing and navigation.
 */
import type { TextRange } from '@numen/plugin-sdk'

export type { TextRange }

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

/** Document layout manifest: page dimensions and disk file stat. */
export interface DocumentLayout {
  readonly pages: readonly Page[]
  readonly stat: string
}

/** Port for document service interactions. */
export interface Documents {
  /** Retrieves document layout and dimensions. */
  getDocumentLayout(path: string): Promise<DocumentLayout>
  /** Returns the image URL for a rendered page at the given pixel width. */
  getPageUrl(path: string, pageNumber: number, width: number, stat?: string): string
  /** Retrieves highlights for given text spans. */
  getHighlights(
    path: string,
    spans: readonly TextRange[],
  ): Promise<readonly (readonly PageHighlight[])[]>
}

/** Port for measuring an attached page view, and for turning its pages. */
export interface PageHandle {
  measure(): void
  /** A key the tab caught: true where it turned the page. */
  handleKey(event: KeyboardEvent): boolean
  /** Gives the pages keyboard focus, so key presses reach them. */
  focusPages(): void
}

/**
 * How a document is drawn when it opens: a whole page in the viewport, a page as wide
 * as the viewport, or a page at its printed size.
 */
export type Fit = 'page' | 'width' | '100%'
