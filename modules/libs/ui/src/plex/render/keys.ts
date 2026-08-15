/** Enter or the space bar: what a keyboard means by pressing something. */
export const isPress = (event: KeyboardEvent): boolean =>
  event.key === 'Enter' || event.key === ' '
