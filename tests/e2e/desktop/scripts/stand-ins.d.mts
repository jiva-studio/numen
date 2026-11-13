/** What the stand-in chat API answers every prompt with. */
export const STAND_IN_REPLY: string
/** The prompt the stand-in answers only after `DELETE /stand-in/hold` opens the gate `PUT` closed. */
export const HELD_PROMPT: string
/** The model the stand-in lists. */
export const STAND_IN_MODEL: string
/** The connection row of the stand-in model. */
export const STAND_IN_CONNECTION: string
/** The prompt the stand-in answers with a note_write tool call. */
export const TOOL_PROMPT: string
/** The path of the note the stand-in writes with that call. */
export const TOOL_NOTE_PATH: string
/** The text of the note the stand-in writes with that call. */
export const TOOL_NOTE_TEXT: string
/** Starts the stand-ins on loopback, and gives back the addresses they answer at. */
export const startStandIns: () => Promise<{
  updateUrl: string
  chatUrl: string
  close: () => void
}>
