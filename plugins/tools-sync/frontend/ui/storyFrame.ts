const FRAME =
  'box-sizing:border-box;inline-size:56rem;block-size:40rem;padding:1rem;overflow:auto;' +
  'transform:translateZ(0);background:var(--numen-surface);color:var(--numen-ink)'

/** Draws a story in a frame as large as a tab, so a dialog covers the frame and not the whole page. */
export const frameDecorator = () => ({
  template: `<div style="${FRAME}"><story /></div>`,
})
