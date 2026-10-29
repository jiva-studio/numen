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
  // Handles the queries made in the view. A plugin passes a port that sends the subject of each query
  // with it, such as the current card.
  readonly agent: AgentPort
  // The vault path the questions are about, sent as their focus.
  readonly focus: string
  // Placeholder text of the empty field; it names the field.
  readonly placeholder?: string
  // Why no query can be made; empty while one can. The view shows it in place of the answers.
  readonly getUnreachable?: () => string
}

// VAULT_SUMMARY is the capability of a view shown next to a vault on the list a window offers to
// choose from, such as what the vault has due today. A window mounts one for each vault on its list.
export const VAULT_SUMMARY = 'vault-summary'

// VaultSummaryContext is what a window passes the vault summary view it mounts.
export interface VaultSummaryContext {
  // The identity of the vault the row represents.
  readonly vault: string
}

// NOTE_EDITOR is the capability of a view that shows one note of the vault and edits it.
export const NOTE_EDITOR = 'note-editor'

// NoteEditorContext is what a plugin or a window passes the note editor view it mounts.
export interface NoteEditorContext {
  // The vault path of the note.
  readonly path: string
}
