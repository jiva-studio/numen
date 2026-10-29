/** The strings of a recording tab: the player, the transcript, and the actions available on it. */
export const WORDS = {
  /** The command that opens a tab to record in. */
  record: 'Record audio',
  player: 'Recording',
  transcript: 'Transcript',
  /** The view scrolls to keep the current line visible. */
  follow: 'Follow',
  /** The menu at the end of the strip, and what it is announced as. */
  more: 'More',
  /** The recording is being read, and what it holds is not known yet. */
  loading: 'Loading…',
  /** Nothing has transcribed this recording. */
  silence: 'No transcript yet.',
  transcribe: 'Transcribe',
  proofread: 'Proofread transcript',
  deleteText: 'Delete transcript',
  /** A run is in progress, and more words arrive as they are transcribed. */
  transcribing: 'Transcribing…',
  /** This window cannot play this kind of audio. */
  unplayable: 'Audio playback not supported here.',
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

/** The ids of what the tab's menu offers. */
export const TRANSCRIBE = 'transcribeText'
export const PROOFREAD = 'proofreadText'
export const DELETE_TEXT = 'deleteText'
