/**
 * Public exports for the right-click menu that moves the entries of the bars.
 */
export {
  useSlotMoveMenu,
  type SlotMoveMenuControl,
  type SlotMoveMenuDeps,
} from './model/useSlotMoveMenu'
export { MOVE_WORDS, getSlotTitle } from './words'
export { default as SlotMoveMenu } from './ui/SlotMoveMenu.vue'
