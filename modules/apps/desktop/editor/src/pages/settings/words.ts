/** What the settings tab says: the groups it draws, and what each setting is. */
export const WORDS = {
  settings: 'Settings',
  /**
   * Where the settings stand, said once at the head of the page. The file
   * itself is named where the vault has said where it is.
   */
  file: 'One file, numen.json, in this machine’s configuration folder.',
  /** The one way to the file itself, which every setting stands in. */
  opens: 'Open the file',
  /** The groups, in the order they are drawn. */
  window: 'The window',
  naming: 'What a note is called',
  review: 'A day of review',
  transcription: 'Transcribing recordings',
  ocr: 'OCR',
  indexing: 'Indexing',
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
  /** Transcribing a recording, and putting the transcript right. */
  transcribing: 'Transcribe recordings',
  transcribingDetail:
    'Whether a recording the vault holds no transcript for is listened to unasked.',
  transcribeUnder: 'Largest recording transcribed',
  transcribeUnderDetail:
    'How large a recording may be, in megabytes, and still be listened to unasked. Below nothing is no limit.',
  transcriptProofread: 'Proofreading a transcript',
  transcriptProofreadDetail: 'The profile a transcript of a recording is put right at.',
  transcriptProofreadAlways: 'Proofread every transcript',
  transcriptProofreadAlwaysDetail:
    'Whether a transcript is put right as it is made, without being asked.',
  /** Reading a scanned page, and putting the reading right. */
  ocrModel: 'OCR model',
  ocrModelDetail: 'The model a scanned page is read by.',
  ocrProofread: 'Proofreading a scanned reading',
  ocrProofreadDetail: 'The profile a reading taken off a scanned page is put right at.',
  ocrProofreadAlways: 'Proofread every scanned reading',
  ocrProofreadAlwaysDetail: 'Whether a reading is put right as it is taken, without being asked.',
  /** The profile naming none, which is what proofreads nothing. */
  proofreadingNone: 'Nothing',
  /** Turning the vault into vectors. */
  indexingModel: 'Embedding model',
  indexingModelDetail: 'The model the text of the vault is turned into vectors by.',
  /** The agent. */
  agentUse: 'Which agent answers',
  agentUseDetail: 'The program the panel asks. Nothing answers where none is named.',
  agentModel: 'The agent’s model',
  agentModelDetail: 'Which of that program’s models answers.',
  agentSteps: 'Steps in a turn',
  agentStepsDetail: 'How many times it may act before it has to answer.',
  agentTools: 'MCP server',
  agentToolsDetail:
    'Whether this vault’s tools are served over MCP, so another program — an editor, a coding agent — can reach them.',
  agentHooks: 'Read hooks and skills',
  agentHooksDetail: 'Whether it reads the hooks and skills this machine is set up with.',
  /** What is said on the row an installation nobody has configured runs on. */
  byDefault: 'the default',
  /**
   * What a model's files are on this machine. A model reached over the network
   * is fetched from nowhere, and stands with neither of these said about it.
   */
  present: 'on this machine',
  notFetched: 'not fetched yet',
}
