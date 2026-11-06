/**
 * What a link tab reads and does, as the plugin's backend and the window serve it.
 */

/** One link: the address its note holds and the addresses of its media and of its player's page. */
export interface LinkFacts {
  readonly url: string
  /** Whether a video is published at the address, so a copy of it can be made. */
  readonly isCopyable: boolean
  /** Whether the text was fetched once and its version deleted since. */
  readonly isTextDeleted: boolean
  /** The address of the downloaded media the note holds, and empty where it holds none. */
  readonly media: string
  /** The address of the page that frames the player of the video, and empty where none is known. */
  readonly page: string
}

/** Where a copy is played from, and what it is played as. */
export interface LinkSource {
  readonly mediaUrl: string
  readonly mediaType: string
}

/** A run that gave an answer, or why it did not. */
export type Outcome<T> =
  { readonly ok: true; readonly value: T } | { readonly ok: false; readonly error: string }

/** What the tab works a link through. */
export interface Links {
  getLink(path: string): Promise<Outcome<LinkFacts>>
  readArticle(path: string): Promise<string>
  /** Where the file at an address is loaded from, and what it is played as. */
  getSource(address: string): Promise<LinkSource>
  /**
   * Starts fetching the text. A fetch that is not forced does not restore a deleted text, and takes
   * a stored text without fetching it.
   */
  fetchText(path: string, isForced: boolean): Promise<Outcome<null>>
  copyVideo(path: string): Promise<Outcome<null>>
  deleteCopy(path: string): Promise<Outcome<null>>
  /**
   * Calls onChange with whether work on the file is running; the returned
   * function stops the notifications.
   */
  watchWork(path: string, onChange: (isWorking: boolean) => void): () => void
}
