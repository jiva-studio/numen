import { defineComponent } from 'vue'
import type { AgentPort } from '@numen/plugin-sdk'

/** A component that draws nothing, for a registration whose component a test never mounts. */
export const STUB_COMPONENT = defineComponent({ name: 'StubComponent', render: () => null })

/** An agent that answers nothing, for a view that needs one to be mounted. */
export const STUB_AGENT: AgentPort = {
  ask: () => ({
    [Symbol.asyncIterator]: () => ({
      next: () => Promise.resolve({ done: true, value: undefined }),
    }),
  }),
  finish: () => Promise.resolve(),
}
