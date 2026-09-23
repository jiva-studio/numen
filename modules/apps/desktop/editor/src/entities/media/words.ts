/**
 * What a recording tab asks for, under the identities the commands give them:
 * the words written down, the words put right, and the words taken away.
 */
export const TRANSCRIBE = 'transcribe'
export const PROOFREAD = 'proofread'
export const DELETE_TEXT = 'deleteText'

/** What a recording tab says: the player, and the transcript of the recording. */
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
  /** Nothing has been fetched for a url, which is where its words come from. */
  unfetched: 'Nothing has been fetched from this address yet.',
  /** The run that writes the words of the recording down, asked for here. */
  transcribe: 'Transcribe',
  /** The words written down, put right by a proofreader, asked for here. */
  proofread: 'Proofread transcript',
  /** The words written down, taken away, asked for here where they stand. */
  deleteText: 'Delete transcript',
  /** A run is going, and more words arrive as they are written down. */
  transcribing: 'Still transcribing…',
  /**
   * This window has nothing to play sound with. It says so and no more: the
   * words of the recording are the transcript's to say.
   */
  unplayable: 'This system cannot play sound here.',
  /** The frame what a url points at plays in, and the bar that resizes it. */
  playing: 'What this url points at',
  taller: 'Drag to resize the player',
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
