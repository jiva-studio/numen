/**
 * What the importer's frontend reads and does, as the plugin's backend serves it.
 */

/** One link: the address its note holds and whether a video is published there. */
export interface LinkFacts {
  readonly url: string
  /** Whether a video is published at the address, so a copy of it can be made. */
  readonly isCopyable: boolean
}

/** A run that gave an answer, or why it did not. */
export type Outcome<T> =
  { readonly ok: true; readonly value: T } | { readonly ok: false; readonly error: string }

/** What the frontend works a link through. */
export interface Links {
  getLink(path: string): Promise<Outcome<LinkFacts>>
  /** Starts downloading the video of a link, or its sound alone. */
  copyMedia(path: string, kind: 'video' | 'audio'): Promise<Outcome<null>>
}
