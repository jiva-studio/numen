const TABBABLE =
  'input, textarea, select, button, [href], [role="combobox"], [role="switch"], [tabindex], [contenteditable="true"]'

/** The elements inside the root that Tab reaches, in document order. */
export function getTabbables(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(TABBABLE)).filter(
    (one) => !one.hasAttribute('disabled') && one.getAttribute('tabindex') !== '-1',
  )
}
