import { doc } from '@numen/ui'
import { createMediaPlayer, type Player } from './mediaPlayer'

/** The one player this window has. Every recording and video is played through it. */
export const player: Player = createMediaPlayer()

/** Reports whether a media type can be played. */
export type MediaTypeProbe = (type: string) => boolean

const probeWindow: MediaTypeProbe = (type) => {
  try {
    const mediaType = type === 'audio/vnd.wave' ? 'audio/wav' : type
    const kind = mediaType.startsWith('video/') ? 'video' : 'audio'
    return doc.createElement(kind).canPlayType(mediaType) !== ''
  } catch {
    // A probe that throws counts as unsupported.
    return false
  }
}

/**
 * Reports whether this window can play a media type. Answers come from the window and are
 * cached per type, regardless of how many recordings are open.
 */
export function createMediaTypeProbe(answers: MediaTypeProbe = probeWindow): MediaTypeProbe {
  const answersByType = new Map<string, boolean>()
  return (type) => {
    if (!type) return false
    const answer = answersByType.get(type)
    if (answer !== undefined) return answer
    const can = answers(type)
    answersByType.set(type, can)
    return can
  }
}
