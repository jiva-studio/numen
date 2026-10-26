import type { AgentPort } from './agent'
import type { Disposable } from './types'

// ViewRegistration declares a view another plugin mounts by its capability, without knowing which
// plugin draws it.
export interface ViewRegistration {
  // What the view does, as the plugin mounting one names it, such as CONVERSATION.
  readonly capability: string
  // The component drawn. Its `context` prop is the context the mounting plugin passes.
  readonly component: unknown
}

// MountedView is a view of another plugin drawn into an element of the plugin that mounted it.
// Disposing it removes the view and releases what the view holds.
export interface MountedView extends Disposable {
  // Moves the keyboard into the view, where the view takes it.
  focus(): void
}

// CONVERSATION is the capability of a view that holds one conversation with an agent.
export const CONVERSATION = 'conversation'

// ConversationContext is what a plugin passes the conversation view it mounts.
export interface ConversationContext {
  // What answers the questions asked in the view. A plugin passes a port that sends what the
  // questions are about beside each one, such as the card in front of the person.
  readonly agent: AgentPort
  // The vault path the questions are about, sent as their focus.
  readonly focus: string
  // What the empty field says; it names the field.
  readonly placeholder?: string
  // Why nothing can be asked, empty while something can. The view shows it where the answers stand.
  readonly getUnreachable?: () => string
}
