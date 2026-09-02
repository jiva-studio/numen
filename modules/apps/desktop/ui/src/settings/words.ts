/** What the settings tab says: the groups it draws, and what each setting is. */
export const WORDS = {
  settings: 'Settings',
  /**
   * Where the settings stand, said once at the head of the page. The file
   * itself is named where the vault has said where it is.
   */
  file: 'One file, numen.json, in this machine’s configuration folder.',
  /** The groups, in the order they are drawn. */
  window: 'The window',
  naming: 'What a note is called',
  review: 'A day of review',
  indexing: 'Reading the vault',
  agent: 'The agent',
  /** The window's own settings. */
  theme: 'Theme',
  themeDetail: 'The stylesheet the window wears.',
  shipped: 'Ships with numen',
  owned: 'Yours',
  mode: 'Light or dark',
  modeDetail: 'Which half of a colour pair every token is read as.',
  pinned: 'The theme worn declares light and dark itself.',
  system: 'System',
  light: 'Light',
  dark: 'Dark',
  interfaceScale: 'Interface size',
  interfaceScaleDetail:
    'How large the chrome, the controls and the type in them are drawn. 1 is as designed.',
  textScale: 'Reading size',
  textScaleDetail: 'How large the text a person reads is set. 1 is as designed.',
  hanging: 'Parts under a node',
  hangingDetail: 'Whether a node hangs the headings of its note under its box.',
  parts: 'How many parts',
  partsDetail: 'How many of them stand under a node at once.',
  syncing: 'Sync title and filename',
  syncingDetail: 'Whether renaming either of the two brings the other into line.',
  dayStarts: 'A day begins at',
  dayStartsDetail:
    'An answer given before this hour is written into the day before. Noon at the latest.',
  /** Reading the vault: the three steps, and what each runs against. */
  indexingModel: 'Indexing',
  indexingModelDetail: 'The model the text of the vault is turned into vectors by.',
  indexingSection: 'Indexing, in full',
  indexingSectionDetail: 'The model, the two stations that make its vectors, and the floor.',
  ocr: 'OCR',
  ocrDetail: 'The model a scanned page is read by.',
  ocrSection: 'OCR, in full',
  ocrSectionDetail: 'The three models a page goes through, and how it is cut into regions.',
  proofreading: 'Proofreading',
  proofreadingDetail: 'The profile a reading is put right at. Naming none proofreads nothing.',
  proofreadingNone: 'Nothing',
  proofreadingSection: 'Proofreading profiles',
  proofreadingSectionDetail: 'Each profile: what it is reached through, its model, and its key.',
  transcribing: 'Transcribe recordings',
  transcribingDetail: 'Whether a recording the vault holds no transcript for is listened to unasked.',
  transcribeUnder: 'Only under',
  transcribeUnderDetail: 'How large such a recording may be, in megabytes.',
  transcriptionSection: 'Transcription, in full',
  transcriptionSectionDetail: 'The models that listen, where they came from, and how speech is found.',
  /** The agent. */
  agentUse: 'Which agent answers',
  agentUseDetail: 'The program the panel asks. Nothing answers where none is named.',
  agentModel: 'Model',
  agentModelDetail: 'Which of that program’s models answers.',
  agentSteps: 'Steps in a turn',
  agentStepsDetail: 'How many times it may act before it has to answer.',
  agentTools: 'Tools on a port',
  agentToolsDetail: 'Whether the vault’s tools are served over HTTP for another program to reach.',
  agentHooks: 'Read hooks and skills',
  agentHooksDetail: 'Whether it reads the hooks and skills this machine is set up with.',
  agentCommand: 'The command',
  agentCommandDetail: 'What is run, and the arguments before the ones numen adds.',
  /** The pencil, and the editor it opens. */
  edit: 'Edit',
  editing: 'Written as JSON5: comments and a comma after the last member are read.',
  keep: 'Keep',
  cancel: 'Cancel',
  unreadable: 'That is not JSON5:',
  /** What is said on the row an installation nobody has configured runs on. */
  byDefault: 'the default',
  /** A model or a program the file names and this build does not offer. */
  notFound: 'not found',
}
