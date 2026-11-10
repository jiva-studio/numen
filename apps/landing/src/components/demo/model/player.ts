import { ref } from 'vue'

/** The playback of one audio file: where it is, how long it is, and whether it plays. */
export function createPlayer() {
  const position = ref(0)
  const duration = ref(0)
  const isPlaying = ref(false)
  let audio: HTMLAudioElement | null = null

  function load(url: string) {
    if (audio) return
    const element = new Audio()
    element.preload = 'metadata'
    element.addEventListener('timeupdate', () => {
      position.value = Math.round(element.currentTime * 1000)
    })
    element.addEventListener('durationchange', () => {
      duration.value = Number.isFinite(element.duration) ? Math.round(element.duration * 1000) : 0
    })
    element.addEventListener('play', () => {
      isPlaying.value = true
    })
    element.addEventListener('pause', () => {
      isPlaying.value = false
    })
    element.src = url
    audio = element
  }

  const pause = () => audio?.pause()

  return {
    position,
    duration,
    isPlaying,
    load,
    pause,
    seek(ms: number) {
      if (audio) audio.currentTime = Math.max(0, ms) / 1000
      position.value = Math.max(0, Math.round(ms))
    },
    toggle() {
      if (!audio) return
      if (audio.paused) audio.play().catch(pause)
      else pause()
    },
  }
}

export type Player = ReturnType<typeof createPlayer>
