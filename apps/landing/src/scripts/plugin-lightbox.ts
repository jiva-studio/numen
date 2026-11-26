const ARROW_STEP: Readonly<Record<string, number>> = { ArrowRight: 1, ArrowLeft: -1 }

/**
 * Wires the screenshot lightbox of a plugin page: a thumbnail opens it at its screenshot, the
 * arrows, the strip and the arrow keys move between screenshots, and the close button, Escape or a
 * click beside the screenshot close it and return focus to the thumbnail.
 */
export function initPluginLightbox(): void {
  const dialog = document.querySelector<HTMLDialogElement>('[data-plugin-lightbox]')
  if (!dialog) return
  const slides = [...dialog.querySelectorAll<HTMLElement>('[data-lightbox-slide]')]
  const thumbs = [...dialog.querySelectorAll<HTMLElement>('[data-lightbox-thumb]')]
  const counter = dialog.querySelector<HTMLElement>('[data-lightbox-counter]')
  let current = 0
  let opener: HTMLElement | null = null

  const show = (index: number): void => {
    current = (index + slides.length) % slides.length
    for (const [one, slide] of slides.entries()) slide.hidden = one !== current
    for (const [one, thumb] of thumbs.entries()) {
      thumb.setAttribute('aria-current', String(one === current))
    }
    if (counter) counter.textContent = `${current + 1} / ${slides.length}`
  }

  for (const shot of document.querySelectorAll<HTMLAnchorElement>('[data-shot]')) {
    shot.addEventListener('click', (event) => {
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.button !== 0) return
      event.preventDefault()
      opener = shot
      show(Number(shot.dataset.shot))
      dialog.showModal()
      dialog.querySelector<HTMLElement>('[data-lightbox-close]')?.focus()
    })
  }

  for (const button of dialog.querySelectorAll<HTMLElement>('[data-lightbox-step]')) {
    button.addEventListener('click', () => show(current + Number(button.dataset.lightboxStep)))
  }
  for (const thumb of thumbs) {
    thumb.addEventListener('click', () => show(Number(thumb.dataset.lightboxThumb)))
  }
  dialog.querySelector('[data-lightbox-close]')?.addEventListener('click', () => dialog.close())

  dialog.addEventListener('click', (event) => {
    const target = event.target
    if (!(target instanceof HTMLElement)) return
    if (target === dialog || target.matches('[data-lightbox-outside], [data-lightbox-slide]')) {
      dialog.close()
    }
  })
  dialog.addEventListener('keydown', (event) => {
    const step = ARROW_STEP[event.key]
    if (step === undefined || slides.length < 2) return
    event.preventDefault()
    show(current + step)
  })
  dialog.addEventListener('close', () => {
    opener?.focus()
    opener = null
  })
}
