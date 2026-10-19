/** What the settings tab says: the groups it draws, and what each setting is. */
// Stryker disable StringLiteral: interface copy
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
  performance: 'Performance',
  agent: 'The agent',
  ai: 'AI',
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
  nesting: 'Nest headings under notes',
  nestingDetail: 'Whether a node nests the headings of its note under its box.',
  headings: 'How many headings',
  headingsDetail: 'How many of them stand under a node at once.',
  syncing: 'Sync title and filename',
  syncingDetail: 'Whether renaming either of the two brings the other into line.',
  dayStartTime: 'A day begins at',
  dayStartTimeDetail:
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
  /** Recognising a scanned page, and putting the recognition right. */
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
  /** Workload and performance calibration. */
  performanceProfile: 'Performance profile',
  performanceProfileDetail:
    'Calibrates compute throughput and hardware saturation across local AI pipelines.',
  eco: 'Eco',
  ecoDetail: 'Minimal power, quiet fans (1 worker thread).',
  balanced: 'Balanced',
  balancedDetail: 'Standard workload, 50% CPU capacity.',
  maximum: 'Maximum',
  maximumDetail: 'Full hardware saturation across all CPU cores and memory.',
  custom: 'Custom',
  customDetail: 'Manual parameter tuning.',
  embeddingBatchSize: 'Embedding batch size',
  embeddingBatchSizeDetail: 'Number of text chunks processed in a single forward pass.',
  ocrThreads: 'OCR parallel threads',
  ocrThreadsDetail: 'Number of parallel CPU worker threads for page rendering.',
  ocrSessions: 'OCR recognition sessions',
  ocrSessionsDetail: 'Number of parallel ONNX text recognition sessions.',
  llmConcurrency: 'LLM concurrency',
  llmConcurrencyDetail: 'Maximum number of concurrent in-flight requests to the proofreader.',
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
  /** Which agent answers the plugins that ask a model, and with which model. */
  defaultAgent: 'Default agent',
  defaultAgentDetail:
    'The agent that answers a plugin whose own setting names none. Left to itself, the installed agent with the lowest id answers.',
  defaultModel: 'Default model',
  defaultModelDetail: 'Which of that agent’s models answers.',
  pluginModel: 'Model',
  agentByDefault: (name: string) => `Default (${name})`,
  agentFirst: (name: string) => `First installed (${name})`,
  agentNone: 'No agent is installed',
  agentMissing: 'not installed: questions are refused, and no other agent answers',
  modelByAgent: 'Agent default',
  /** Whether a plugin may ask a model at all. */
  consentAllowed: 'May ask a model',
  consentDetail: {
    none: '',
    unanswered: 'Not asked yet. Its first question is refused until you choose.',
    asked: 'Asked to use a model and was refused until you choose.',
    allowed: 'Allowed: its questions go to the agent its setting names.',
    refused: 'Refused: its questions are answered with an error it shows.',
  },
  asking: (name: string) => `${name} asks to use a model.`,
  allow: 'Allow',
  refuse: 'Refuse',
  /** The plugins the host suspends after they failed to start. */
  pluginsStopped: 'Plugins that stopped',
  pluginStoppedAt: (when: string) => `Stopped ${when}`,
  pluginRetry: 'Retry',
  pluginDisable: 'Disable',
  pluginRetryFor: (name: string) => `Retry ${name}`,
  pluginDisableFor: (name: string) => `Disable ${name}`,
  /** The keys of the secrets plugins declare. */
  keys: 'Keys',
  keyStored: 'Set',
  keyMissing: 'Not set',
  keyDetail: 'Stored in the system keychain, never in the settings file, and never shown again.',
  keyPlaceholder: 'Paste a key',
  keySave: 'Save',
  keyReplace: 'Replace',
  keyDelete: 'Delete',
  /** What is said on the row an installation nobody has configured runs on. */
  byDefault: 'the default',
  /**
   * What a model's files are on this machine. A model reached over the network
   * is fetched from nowhere, and stands with neither of these said about it.
   */
  present: 'on this machine',
  notFetched: 'not fetched yet',
}
