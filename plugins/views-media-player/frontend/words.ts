/** What a recording tab says: the player, the transcript, and what can be asked about it. */
export const WORDS = {
  player: 'The recording',
  transcript: 'The transcript',
  /** The view keeps the line being said in sight. */
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
  /** A run is going, and more words arrive as they are written down. */
  transcribing: 'Still transcribing…',
  /** This window has nothing to play this kind of sound with. */
  unplayable: 'This system cannot play sound here.',
  /** The player stopped and said nothing a code names. */
  unreadable: 'The recording could not be played.',
  /** The person, or the window, stopped the loading. */
  stopped: 'Loading the recording was stopped.',
  /** The bytes never arrived. */
  unreached: 'The recording could not be read from the vault.',
  /** They arrived and are not what they claim, or nothing here decodes them. */
  undecoded: 'The recording is in a form this system cannot decode.',
  /** The player refused the type before asking for a byte of it. */
  unwanted: 'This system does not play recordings of this kind.',
}

/** The ids of what the tab's menu offers. */
export const PROOFREAD = 'proofread'
export const DELETE_TEXT = 'deleteText'
