/**
 * Asking an agent about the vault the window is showing.
 *
 * The window hands over what was written and reads what the agent does as it
 * does it. What answers sits behind this port, the way the vault sits behind
 * `vault.ts`.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { AgentService } from '@numen/protocol'

/** One thing the agent said, did, or stopped for. */
export type Step =
  | { readonly kind: 'said'; readonly text: string }
  | {
      readonly kind: 'doing'
      readonly tool: string
      readonly about: string
      /** How much of the call has been written. It arrives more than once. */
      readonly written: number
    }
  /** The tool answered. Nothing of this application's is running from here. */
  | { readonly kind: 'answered' }
  /** A request to the model has begun: this is where a wait starts. */
  | { readonly kind: 'thinking' }
  | { readonly kind: 'stopped'; readonly failed: string }

export interface Agent {
  /**
   * Ask about the note in focus, and read what happens as it happens.
   *
   * The conversation is which thread of talk the question belongs to. Questions
   * carrying one name are answered as one conversation, and a name stands for
   * one conversation and is never given to a second.
   */
  readonly ask: (
    asked: string,
    focus: string,
    conversation: string,
    signal: AbortSignal,
  ) => AsyncIterable<Step>

  /**
   * Say a conversation is over. What the agent kept of it is let go of, and
   * whatever is still being answered in it stops.
   */
  readonly finish: (conversation: string) => Promise<void>
}

const agent = createClient(
  AgentService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

export const core: Agent = {
  async *ask(asked, focus, conversation, signal) {
    for await (const step of agent.ask({ asked, focus, conversation }, { signal })) {
      switch (step.step.case) {
        case 'said':
          yield { kind: 'said', text: step.step.value }
          break
        case 'doing':
          yield {
            kind: 'doing',
            tool: step.step.value.tool,
            about: step.step.value.about,
            written: step.step.value.written,
          }
          break
        case 'answered':
          yield { kind: 'answered' }
          break
        case 'thinking':
          yield { kind: 'thinking' }
          break
        case 'stopped':
          yield { kind: 'stopped', failed: step.step.value }
          break
      }
    }
  },

  async finish(conversation) {
    await agent.finish({ conversation })
  },
}
