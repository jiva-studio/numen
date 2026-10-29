/** Enter or the space bar: what a keyboard means by pressing something. */
export const isPress = (event: KeyboardEvent): boolean => event.key === 'Enter' || event.key === ' '

/**
 * What a keyboard means by requesting a node to be expanded: a press with
 * Shift held. Alt selects where it goes, as it does with a pointer.
 */
export const isShowKey = (event: KeyboardEvent): boolean => event.shiftKey && isPress(event)

/**
 * What a keyboard means by asking for a menu: Shift+F10, and the key some
 * keyboards carry for it.
 */
export const isMenuKey = (event: KeyboardEvent): boolean =>
  event.key === 'ContextMenu' || (event.shiftKey && event.key === 'F10')
