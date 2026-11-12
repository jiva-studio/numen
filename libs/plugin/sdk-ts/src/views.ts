import type { Component } from 'vue'
import type { ConnectionHost } from './connections'
import type { Disposable } from './types'

// ViewRegistration declares a view another plugin mounts by its capability, without knowing which
// plugin draws it. The manifest lists the capability under `contributes.viewCapabilities`, only one
// plugin draws a capability, and the plugin holds the permissions its context gives the view
// (note-editor: vault:read and vault:write; vault-summary: vault:read; conversation: none).
// Otherwise `register.view` throws.
export interface ViewRegistration {
  readonly capability: ViewCapability
  readonly component: Component
}

// ViewContexts is what a view of each capability is passed when it is mounted.
export interface ViewContexts {
  readonly conversation: ConversationContext
  readonly 'vault-summary': VaultSummaryContext
  readonly 'note-editor': NoteEditorContext
}

// ViewCapability names a kind of view that one plugin draws and another mounts.
export type ViewCapability = keyof ViewContexts

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
  // The connections of the mounting plugin the view runs its tasks on.
  readonly connections: ConnectionHost
  // The keys the tools of a task are called with.
  readonly toolContext?: Readonly<Record<string, string>>
  readonly vault?: string
  readonly placeholder?: string
}

// VAULT_SUMMARY is the capability of a view shown next to a vault on the list a window offers to
// choose from, such as what the vault has due today. A window mounts one for each vault on its list.
export const VAULT_SUMMARY = 'vault-summary'

// VaultSummaryContext is what a window passes the vault summary view it mounts.
export interface VaultSummaryContext {
  readonly vault: string
}

// NOTE_EDITOR is the capability of a view that shows one note of the vault and edits it.
export const NOTE_EDITOR = 'note-editor'

// NoteEditorContext is what a plugin or a window passes the note editor view it mounts.
export interface NoteEditorContext {
  readonly path: string
}
