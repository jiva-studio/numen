const FRAME =
  'position:relative;box-sizing:border-box;inline-size:100vw;block-size:100vh;padding:1rem;overflow:auto;' +
  'background:var(--numen-surface);color:var(--numen-ink)'

/** Draws a story in a frame as large as the window, where every dialog is centred under one backdrop. */
export const frameDecorator = () => ({
  template: `<div class="numen" style="${FRAME}"><story /></div>`,
})
