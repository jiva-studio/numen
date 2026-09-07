/** The runs a note tab offers over the address its note points at. */
export const FETCH = 'fetch'
export const DOWNLOAD = 'download'
export const DROP = 'dropTranscript'

/** What a note tab says: the questions its file puts, and the two ways out. */
export const WORDS = {
  overtaken: 'The file changed on disk, so this note stopped saving.',
  gone: 'This note is no longer in the vault, so saving stopped. What is here is still yours.',
  makeAgain: 'make it again',
  keep: 'Keep mine',
  take: "Take the file's",
  newNote: 'New note',
  /** What is at the address a link note points at, drawn over its prose. */
  playing: 'What this note points at',
  /** What the words fetched for an address are called, for whoever is not reading them. */
  transcript: 'The transcript',
  /** What the menu over the player offers, and what it is called. */
  more: 'What can be asked about this',
  fetch: 'Fetch what is at this address',
  download: 'Download a copy of this video',
  drop: 'Delete the transcript',
  /** What stands where the transcript would, until something is fetched. */
  nothingFetched: 'Nothing has been fetched from this address yet.',
  /** The bar that draws the player taller or shorter. */
  taller: 'Drag to resize the player',
}
