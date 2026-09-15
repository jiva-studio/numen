/** A shared module reaching the two layers above it, which is refused twice. */
import { deck } from '../../features/cards'
import { agent } from '../../screens/agent'

export const sorted = () => `${deck()} ${agent()}`
