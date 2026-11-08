/** The strings of a recording tab: the player, the text, and the actions available on it. */
export const WORDS = {
  player: 'Recording',
  /** The page that frames the player of a video. */
  page: 'Video player',
  transcript: 'Transcript',
  /** The view scrolls to keep the current line visible. */
  follow: 'Follow',
  /** The button that creates the text of a recording nothing was heard in. */
  transcribe: 'Transcribe',
  /** The recording is being read, and its contents are not known yet. */
  loading: 'Loading…',
  /** Nothing has transcribed this recording. */
  silence: 'No transcript yet.',
  /** A run is in progress, and more words arrive as they are transcribed. */
  transcribing: 'Transcribing…',
  /** This window cannot play this type of audio. */
  unplayable: 'Media playback not supported here.',
  /** The player stopped with no error code. */
  unreadable: 'Recording failed to play.',
  /** The user or the window stopped the loading. */
  stopped: 'Loading stopped.',
  /** The bytes never arrived. */
  unreached: 'Recording unreadable.',
  /** The bytes arrived but are not the claimed format, or no decoder is available. */
  undecoded: 'Recording format not supported.',
  /** The player rejected the type before requesting any data. */
  unwanted: 'Recording type not supported.',
}
