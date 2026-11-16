import type { Voice } from '../../types'

export interface VoiceDescriptor {
  readonly isBubble: boolean
  readonly against: 'start' | 'end'
}

/**
 * Every voice a turn can be in, declared once. The shape, the side and the
 * colours all read from here.
 */
export const VOICES = {
  prompt: { isBubble: true, against: 'end' },
  response: { isBubble: false, against: 'start' },
  doing: { isBubble: false, against: 'start' },
  plan: { isBubble: false, against: 'start' },
} as const satisfies Record<Voice, VoiceDescriptor>
