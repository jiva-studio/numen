/**
 * The full ids of the commands the demo window handles: the commands of the editor under the owner
 * `app`, and those of the plugins it draws under the id of their plugin. The window sends the same
 * ids, so the demo answers what a real window asks.
 */
export const DEMO_COMMAND = {
  openNote: 'app:note.open',
  openNoteBeside: 'app:note.open-beside',
  createNote: 'app:note.create',
  createChildNote: 'app:note.create-child',
  createParentNote: 'app:note.create-parent',
  createJumpNote: 'app:note.create-jump',
  renameNote: 'app:note.rename',
  trashNote: 'app:note.trash',
  copyNotePath: 'app:note.copy-path',
  openFiles: 'md.numen.views.file.tree:tree.open',
  revealNote: 'md.numen.views.file.tree:tree.reveal-note',
  openPlex: 'md.numen.views.plex.graph:graph.open',
  showNoteInPlex: 'md.numen.views.plex.graph:graph.show-note',
  openAgent: 'md.numen.views.agent.chat:chat.open',
  askAgent: 'md.numen.views.agent.chat:note.ask',
} as const

/** The id of the command a transcript menu entry runs to delete a transcript. */
export const DELETE_TRANSCRIPT = 'transcript.delete'
