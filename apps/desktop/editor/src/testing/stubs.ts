/**
 * and not the editor it draws.
 */
import { defineComponent, h } from 'vue'
import { requests } from './requests'

/** A component rendered in a pane that returns the given values for the window's calls. */
const createStub = (name: string, answers: Record<string, unknown>) =>
  defineComponent({
    name,
    setup: (_, { expose }) => {
      expose(answers)
      return () => h('div')
    },
  })

/** An editor stub implements the three methods a note calls on an editor; a page stub implements one. */
export const editor = createStub('Editor', {
  focus: () => true,
  measure: () => (requests.measured += 1),
  reveal: () => true,
})

export const reader = createStub('Reader', {
  measure: () => {},
  // The pages record every key received and do nothing else: the window only has
  // to deliver the key to the active tab.
  handleKey: (event: KeyboardEvent) => {
    requests.pressed.push(event.key)
    return event.key.startsWith('Arrow')
  },
  focusPages: () => {},
})

export const book = createStub('Book', {
  measure: () => {},
  // The book records every key received and turns no page: the window only has to
  // deliver the key to the active book.
  handleKey: (event: KeyboardEvent) => {
    requests.pressed.push(event.key)
    return event.key.startsWith('Arrow')
  },
})
