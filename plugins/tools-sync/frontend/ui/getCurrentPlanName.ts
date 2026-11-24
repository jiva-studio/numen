import { defineComponent, h, markRaw } from 'vue'
import { WORDS as words } from '../words'

/** The menu row of the current plan: its name and the tag "Current plan". */
export const getCurrentPlanName = (name: string) =>
  markRaw(
    defineComponent({
      name: 'CurrentPlanName',
      setup: () => () => [
        h('span', name),
        h('span', { class: 'subscription__current-tag' }, words.currentPlan),
      ],
    }),
  )
