/** What the stand-in chat API answers every prompt with. */
export const STAND_IN_REPLY: string
/** The prompt the stand-in answers only after a pause. */
export const DELAYED_PROMPT: string
/** The model the stand-in lists. */
export const STAND_IN_MODEL: string
/** Starts the stand-ins on loopback, and gives back the addresses they answer at. */
export const startStandIns: () => Promise<{
  updateUrl: string
  chatUrl: string
  close: () => void
}>
