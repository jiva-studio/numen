/**
 * What a link tab reads and does, as the plugin's backend and the window serve it.
 */

/** One link: the address its note holds and the copy stored of its video. */
export interface LinkFacts {
  readonly url: string
  /** Whether a video is published at the address, so a copy of it can be made. */
  readonly isCopyable: boolean
  /** How many bytes the copy holds, and zero where none is kept. */
  readonly copySize: number
  /** Whether the text was fetched once and its version deleted since. */
  readonly isTextDeleted: boolean
  /** The page below the plugin route that frames the player of the video, and empty where none is known. */
  readonly embedPage: string
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
  getSource(path: string): Promise<LinkSource>
  /** Where a page below the plugin route is loaded from in a frame. */
  getPageSource(page: string): Promise<string>
  /** Starts fetching the text. A fetch the person did not ask for leaves a deleted text deleted. */
  fetchText(path: string, isUserStarted: boolean): Promise<Outcome<null>>
  copyVideo(path: string): Promise<Outcome<null>>
  deleteCopy(path: string): Promise<Outcome<null>>
  /** Tells onChange whether work on the file is going, and stops telling once called back. */
  watchWork(path: string, onChange: (isWorking: boolean) => void): () => void
}
