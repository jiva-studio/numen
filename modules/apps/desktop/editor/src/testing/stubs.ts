/**
 * What stands in the panes, so that what is asked of the window is the window
 * and not the editor it draws.
 */
import { defineComponent, h } from 'vue'
import { requests } from './requests'

/** Something drawn in a pane that answers what the window asks of it. */
const createStub = (name: string, answers: Record<string, unknown>) =>
  defineComponent({
    name,
    setup: (_, { expose }) => {
      expose(answers)
      return () => h('div')
    },
  })

/** An editor answers the three things a note asks of one; a page, the one. */
export const editor = createStub('Editor', {
  focus: () => true,
  measure: () => (requests.measured += 1),
  reveal: () => true,
})

export const reader = createStub('Reader', {
  measure: () => {},
  // The pages write down every key they were handed and turn to nothing: what
  // is asked of the window is that the key reaches the tab it is showing.
  handleKey: (event: KeyboardEvent) => {
    requests.pressed.push(event.key)
    return event.key.startsWith('Arrow')
  },
  focusPages: () => {},
})

export const book = createStub('Book', {
  measure: () => {},
  // The book writes down every key it was handed and turns no page: what is
  // asked of the window is that the key reaches the book it is showing.
  handleKey: (event: KeyboardEvent) => {
    requests.pressed.push(event.key)
    return event.key.startsWith('Arrow')
  },
})
