/** What a recording tab says: the player, and the words heard in the recording. */
export const WORDS = {
  player: 'The recording',
  transcript: 'What was heard',
  /** Nothing has listened to this recording. */
  silence: 'No transcript yet.',
  /** A run is going, and more words arrive as they are written down. */
  transcribing: 'Still transcribing…',
  /**
   * This window has nothing to play sound with. It says so and no more: what
   * was heard in the recording is the transcript's to say.
   */
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
