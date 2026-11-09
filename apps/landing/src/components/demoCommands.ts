/**
 * The full ids of the commands the demo window handles: the commands of the editor under the owner
 * `app`, and those of the plugins it draws under the id of their plugin. The window sends the same
 * ids, so the demo answers what a real window asks.
 */
export const DEMO_COMMAND = {
  openNote: 'app:note.open',
  openNoteBeside: 'app:note.openBeside',
  createNote: 'app:note.create',
  createChildNote: 'app:note.createChild',
  createParentNote: 'app:note.createParent',
  createJumpNote: 'app:note.createJump',
  renameNote: 'app:note.rename',
  trashNote: 'app:note.trash',
  copyNotePath: 'app:note.copyPath',
  openFiles: 'md.numen.views.file.tree:tree.open',
  revealNote: 'md.numen.views.file.tree:tree.revealNote',
  openPlex: 'md.numen.views.plex.graph:graph.open',
  showNoteInPlex: 'md.numen.views.plex.graph:graph.showNote',
  openAgent: 'md.numen.views.agent.chat:chat.open',
  askAgent: 'md.numen.views.agent.chat:note.ask',
  newTreeFolder: 'md.numen.views.file.tree:tree.createFolder',
  renameTreeEntry: 'md.numen.views.file.tree:tree.renameFile',
} as const
