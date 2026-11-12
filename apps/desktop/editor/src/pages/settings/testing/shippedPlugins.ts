/**
 * The plugins of the repository as an installation lists them: written from the manifest.json
 * and README.md of every folder under plugins by a script, so a story draws what the application ships.
 */
import type { CatalogEntry } from '@/entities/plugin'
import type { PluginInfo } from '@/entities/settings'

export const SHIPPED_PLUGINS: readonly PluginInfo[] = [
  {
    "id": "md.numen.agents.acp",
    "name": "ACP Agents",
    "version": "2026.11.14",
    "description": "External AI assistants via the Agent Client Protocol: Claude Code, Codex, Antigravity and any other ACP program",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "requestTimeout",
        "type": "number",
        "title": "Request timeout",
        "description": "Seconds to wait for an answer; empty waits 120",
        "min": 1,
        "max": 3600,
        "step": 1
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Connection types",
        "items": [
          {
            "text": "ACP agent",
            "detail": "acp",
            "keys": ""
          },
          {
            "text": "Claude Code",
            "detail": "claude-code",
            "keys": ""
          },
          {
            "text": "OpenAI Codex",
            "detail": "codex",
            "keys": ""
          },
          {
            "text": "Antigravity",
            "detail": "antigravity",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Operations",
        "items": [
          {
            "text": "run agent tasks",
            "detail": "acp.task",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "link:open",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:manage",
            "detail": "",
            "keys": ""
          },
          {
            "text": "window:focus",
            "detail": "",
            "keys": ""
          },
          {
            "text": "index:read",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "numen.v1.ProviderService",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.agents.mcp.server",
    "name": "Model Context Protocol Server",
    "version": "2026.11.15",
    "description": "Connects external AI apps to your open vault",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "port",
        "type": "number",
        "title": "Port",
        "description": "Local port MCP clients connect to",
        "default": 3333,
        "min": 1024,
        "max": 65535,
        "step": 1
      },
      {
        "key": "token",
        "type": "string",
        "title": "Token",
        "description": "Bearer token MCP clients send; made on first start"
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Listeners",
        "items": [
          {
            "text": "mcp",
            "detail": "tcp, port from port",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:manage",
            "detail": "",
            "keys": ""
          },
          {
            "text": "window:focus",
            "detail": "",
            "keys": ""
          },
          {
            "text": "index:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "link:open",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.agents.mcp.server",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.agents.numen",
    "name": "Numen Agent",
    "version": "2026.11.12",
    "description": "Built-in AI assistant that chats through a connection to OpenRouter, OpenAI or an OpenAI-compatible server",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "connections",
        "type": "connections",
        "title": "Connection",
        "description": "Chat service, model and reasoning level",
        "max": 1,
        "operations": [
          "llm.chat"
        ]
      },
      {
        "key": "temperature",
        "type": "range",
        "title": "Temperature",
        "description": "Lower is steadier, higher is more varied",
        "default": 0.7,
        "min": 0,
        "max": 1,
        "step": 0.05
      },
      {
        "key": "maxTokens",
        "type": "number",
        "title": "Max Tokens",
        "description": "Most tokens one answer may have",
        "default": 4096,
        "min": 256,
        "max": 128000,
        "step": 256
      },
      {
        "key": "maxSteps",
        "type": "number",
        "title": "Max Steps",
        "description": "Most model requests one task may make",
        "default": 25,
        "min": 1,
        "max": 1000,
        "step": 1
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Connection types",
        "items": [
          {
            "text": "Numen agent",
            "detail": "numen",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Operations",
        "items": [
          {
            "text": "run agent tasks",
            "detail": "acp.task",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:manage",
            "detail": "",
            "keys": ""
          },
          {
            "text": "window:focus",
            "detail": "",
            "keys": ""
          },
          {
            "text": "index:read",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "numen.v1.ProviderService",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.agents.tools.vault",
    "name": "Vault Knowledge Tools",
    "version": "2026.11.8",
    "description": "Library search, reading, and note editing tools for AI assistants",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [
      "tools"
    ],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:manage",
            "detail": "",
            "keys": ""
          },
          {
            "text": "index:read",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Capabilities",
        "items": [
          {
            "text": "tools",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.agents.tools.window",
    "name": "Window Interaction Tools",
    "version": "2026.11.8",
    "description": "Workspace context and interactive edit previews for AI assistants",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [
      "tools"
    ],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "window:focus",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Capabilities",
        "items": [
          {
            "text": "tools",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.connections.llm",
    "name": "Language model services",
    "version": "2026.11.2",
    "description": "Connects Numen to OpenRouter, OpenAI and OpenAI-compatible servers for completions, chat and embeddings",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "requestTimeout",
        "type": "number",
        "title": "Request timeout",
        "description": "Seconds to wait; empty waits 120, or 60 for embeddings",
        "min": 1,
        "max": 3600,
        "step": 1
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Connection types",
        "items": [
          {
            "text": "OpenRouter",
            "detail": "openrouter",
            "keys": ""
          },
          {
            "text": "OpenAI",
            "detail": "openai",
            "keys": ""
          },
          {
            "text": "OpenAI-compatible server",
            "detail": "openai-compatible",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Operations",
        "items": [
          {
            "text": "complete text",
            "detail": "llm.complete",
            "keys": ""
          },
          {
            "text": "chat",
            "detail": "llm.chat",
            "keys": ""
          },
          {
            "text": "embed text",
            "detail": "llm.embed",
            "keys": ""
          },
          {
            "text": "describe a model",
            "detail": "llm.profile",
            "keys": ""
          },
          {
            "text": "list models",
            "detail": "llm.models",
            "keys": ""
          },
          {
            "text": "list reasoning levels",
            "detail": "llm.reasoning",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "numen.v1.ProviderService",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.engines.embeddings",
    "name": "Vector Embeddings",
    "version": "2026.11.13",
    "description": "Meaning-based search across your notes, books, and recordings",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [
      "search.source"
    ],
    "settings": [
      {
        "key": "source",
        "type": "enum",
        "title": "Source",
        "description": "Where texts are turned into vectors",
        "default": "local",
        "options": [
          {
            "value": "local",
            "title": "Local model",
            "detail": "Runs an embedding model on this machine"
          },
          {
            "value": "connection",
            "title": "Connection",
            "detail": "Sends the texts to a service through a connection"
          }
        ]
      },
      {
        "key": "model",
        "type": "string",
        "title": "Model",
        "description": "Local embedding model; it needs a profile",
        "default": "Xenova/multilingual-e5-small",
        "options": [
          {
            "value": "Xenova/multilingual-e5-small",
            "title": "E5 small (default, CPU, 118 MB)",
            "detail": "384 dims · Hugging Face"
          },
          {
            "value": "onnx-community/harrier-oss-v1-270m-ONNX",
            "title": "Harrier 270M (better, 344 MB)",
            "detail": "640 dims · Hugging Face"
          },
          {
            "value": "Xenova/bge-m3",
            "title": "BGE-M3 (large, 569 MB)",
            "detail": "1024 dims · Hugging Face"
          }
        ],
        "when": {
          "setting": "source",
          "is": [
            "local"
          ]
        }
      },
      {
        "key": "connections",
        "type": "connections",
        "title": "Connection",
        "description": "A new connection or model reindexes the vault",
        "when": {
          "setting": "source",
          "is": [
            "connection"
          ]
        },
        "max": 1,
        "operations": [
          "llm.embed",
          "llm.profile"
        ]
      },
      {
        "key": "minChunkLength",
        "type": "number",
        "title": "Smallest piece of text to index for meaning",
        "description": "Shorter pieces of text get no vector",
        "default": 40,
        "min": 0,
        "max": 1000,
        "step": 1
      },
      {
        "key": "chunkTokens",
        "type": "number",
        "title": "Longest chunk of a note, in tokens",
        "description": "Largest piece of a note that gets one vector",
        "default": 256,
        "min": 16,
        "max": 8192,
        "step": 1
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "link:open",
            "detail": "",
            "keys": ""
          },
          {
            "text": "index:read",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Dependencies",
        "items": [
          {
            "text": "onnxruntime",
            "detail": "runtime",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Capabilities",
        "items": [
          {
            "text": "search.source",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.engines.ocr",
    "name": "Document OCR",
    "version": "2026.11.19",
    "description": "Text recognition for scanned PDFs and images",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [
      "tools"
    ],
    "settings": [
      {
        "key": "model",
        "type": "enum",
        "title": "Model",
        "description": "Text recognition model",
        "default": "pp-ocrv6-tiny",
        "options": [
          {
            "value": "pp-ocrv6-tiny",
            "title": "PP-OCRv6 tiny",
            "detail": "Chinese, English and Latin-script languages, 4.5 MB"
          },
          {
            "value": "pp-ocrv6-small",
            "title": "PP-OCRv6 small",
            "detail": "Chinese, English, Japanese and Latin-script languages, 21 MB"
          },
          {
            "value": "pp-ocrv5-latin",
            "title": "PP-OCRv5 Latin",
            "detail": "English, French, German, Spanish, Italian, Portuguese, Dutch, Polish, Czech, Slovak, Hungarian, Romanian, Croatian, Slovenian, Danish, Swedish, Norwegian, Finnish, Estonian, Lithuanian, Latvian, Turkish, Latin and other Latin-script languages, 7.9 MB"
          },
          {
            "value": "pp-ocrv5-eslav",
            "title": "PP-OCRv5 East Slavic",
            "detail": "Russian, Ukrainian, Belarusian, English, 7.9 MB"
          },
          {
            "value": "pp-ocrv5-cyrillic",
            "title": "PP-OCRv5 Cyrillic",
            "detail": "Russian, Ukrainian, Belarusian, Bulgarian, Serbian, Macedonian, Mongolian, Kazakh, Kyrgyz, Tajik, Tatar and other Cyrillic-script languages, English, 8.1 MB"
          },
          {
            "value": "pp-ocrv5-devanagari",
            "title": "PP-OCRv5 Devanagari",
            "detail": "Hindi, Sanskrit, Marathi, Nepali, Konkani, Maithili, Bhojpuri, English, 7.9 MB"
          },
          {
            "value": "pp-ocrv5-greek",
            "title": "PP-OCRv5 Greek",
            "detail": "Greek, English, 7.8 MB"
          },
          {
            "value": "custom",
            "title": "Custom model",
            "detail": "Languages unknown: they are those of the model file"
          }
        ]
      },
      {
        "key": "customModelPath",
        "type": "file",
        "title": "Model file",
        "description": "PaddleOCR text recognition model in ONNX format",
        "default": "",
        "when": {
          "setting": "model",
          "is": [
            "custom"
          ]
        },
        "accept": [
          ".onnx"
        ]
      },
      {
        "key": "customDictPath",
        "type": "file",
        "title": "Character dictionary",
        "description": "One character per line; empty if the model has its own",
        "default": "",
        "when": {
          "setting": "model",
          "is": [
            "custom"
          ]
        },
        "accept": [
          ".txt"
        ]
      },
      {
        "key": "dpi",
        "type": "enum",
        "title": "Scan resolution",
        "description": "Resolution a page is rendered at before recognition",
        "default": "300",
        "options": [
          {
            "value": "150",
            "title": "150 dpi",
            "detail": ""
          },
          {
            "value": "200",
            "title": "200 dpi",
            "detail": ""
          },
          {
            "value": "300",
            "title": "300 dpi",
            "detail": ""
          },
          {
            "value": "400",
            "title": "400 dpi",
            "detail": ""
          },
          {
            "value": "600",
            "title": "600 dpi",
            "detail": ""
          }
        ]
      },
      {
        "key": "recognizeAutomatically",
        "type": "boolean",
        "title": "Recognize automatically",
        "description": "Recognize every PDF and image that has no text",
        "default": false
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Handlers",
        "items": [
          {
            "text": "Recognize text",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Triggers",
        "items": [
          {
            "text": "file.added",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Text kinds",
        "items": [
          {
            "text": "Recognized text",
            "detail": "recognition",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "fileTree.context",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "link:open",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Dependencies",
        "items": [
          {
            "text": "onnxruntime",
            "detail": "runtime",
            "keys": ""
          },
          {
            "text": "pp-doclayout",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "pp-ocr-det",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "rec-pp-ocrv6-tiny",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "rec-pp-ocrv6-small",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "rec-pp-ocrv5-latin",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "rec-pp-ocrv5-eslav",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "rec-pp-ocrv5-cyrillic",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "rec-pp-ocrv5-devanagari",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "rec-pp-ocrv5-greek",
            "detail": "model",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Capabilities",
        "items": [
          {
            "text": "tools",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.engines.transcription",
    "name": "Audio/Video Transcription",
    "version": "2026.11.19",
    "description": "Offline speech-to-text transcription for audio and video recordings",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [
      "tools"
    ],
    "settings": [
      {
        "key": "model",
        "type": "enum",
        "title": "Model",
        "description": "Parakeet v3 knows 25 languages, v2 English only",
        "default": "parakeet-tdt-0.6b-v3-int8",
        "options": [
          {
            "value": "parakeet-tdt-0.6b-v3-int8",
            "title": "Parakeet TDT 0.6B v3",
            "detail": "25 European languages, 670 MB"
          },
          {
            "value": "parakeet-tdt-0.6b-v2-int8",
            "title": "Parakeet TDT 0.6B v2",
            "detail": "English, 660 MB"
          }
        ]
      },
      {
        "key": "transcribeAutomatically",
        "type": "boolean",
        "title": "Transcribe automatically",
        "description": "Transcribe every recording that has no transcript",
        "default": false
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Handlers",
        "items": [
          {
            "text": "Transcribe",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Triggers",
        "items": [
          {
            "text": "file.added",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Text kinds",
        "items": [
          {
            "text": "Transcript",
            "detail": "transcript",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "fileTree.context",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "link:open",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Dependencies",
        "items": [
          {
            "text": "onnxruntime",
            "detail": "runtime",
            "keys": ""
          },
          {
            "text": "silero-vad",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v3-encoder",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v3-decoder",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v3-joiner",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v3-tokens",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v2-encoder",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v2-decoder",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v2-joiner",
            "detail": "model",
            "keys": ""
          },
          {
            "text": "parakeet-v2-tokens",
            "detail": "model",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Capabilities",
        "items": [
          {
            "text": "tools",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.notes",
    "name": "Notes",
    "version": "2026.11.9",
    "description": "Markdown note editor with live formatting and wikilinks",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "softWrap",
        "type": "boolean",
        "title": "Soft Wrap",
        "description": "Wrap long lines to fit the editor viewport",
        "default": true
      },
      {
        "key": "lineNumbers",
        "type": "boolean",
        "title": "Line Numbers",
        "description": "Show line numbers in the editor margin",
        "default": false
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "Note",
            "detail": "note",
            "keys": ""
          },
          {
            "text": "Markdown Note Editor",
            "detail": "reader:markdown",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Readers",
        "items": [
          {
            "text": "markdown",
            "detail": "text/markdown, text/plain",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "titleBar.commands",
            "detail": "app:note.create",
            "keys": ""
          }
        ]
      },
      {
        "kind": "View capabilities",
        "items": [
          {
            "text": "note-editor",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.readers.epub",
    "name": "EPUB Book Reader",
    "version": "2026.11.16",
    "description": "EPUB book reader with highlights and passage links",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "EPUB Reader",
            "detail": "reader:reader-epub",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Readers",
        "items": [
          {
            "text": "reader-epub",
            "detail": "application/epub+zip",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Handlers",
        "items": [
          {
            "text": "Extract text",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Triggers",
        "items": [
          {
            "text": "file.added",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Text kinds",
        "items": [
          {
            "text": "Book text",
            "detail": "book-text",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "fileTree.context",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.readers.epub",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.readers.pdf",
    "name": "PDF Reader",
    "version": "2026.11.13",
    "description": "PDF document reader with highlights and passage links",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "defaultZoom",
        "type": "enum",
        "title": "Default Zoom",
        "description": "Zoom a PDF opens at",
        "default": "page",
        "options": [
          {
            "value": "page",
            "title": "page",
            "detail": ""
          },
          {
            "value": "width",
            "title": "width",
            "detail": ""
          },
          {
            "value": "100%",
            "title": "100%",
            "detail": ""
          }
        ]
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "PDF Reader",
            "detail": "reader:reader-pdf",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Readers",
        "items": [
          {
            "text": "reader-pdf",
            "detail": "application/pdf",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Handlers",
        "items": [
          {
            "text": "Extract text layer",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Triggers",
        "items": [
          {
            "text": "file.added",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Text kinds",
        "items": [
          {
            "text": "Text layer",
            "detail": "page-text",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "fileTree.context",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.readers.pdf",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.themes.presets",
    "name": "Preset Themes",
    "version": "2026.11.2",
    "description": "Twelve color themes for the editor and the flashcards window",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Themes",
        "items": [
          {
            "text": "Amber",
            "detail": "dark",
            "keys": ""
          },
          {
            "text": "Ayu",
            "detail": "",
            "keys": ""
          },
          {
            "text": "Catppuccin",
            "detail": "",
            "keys": ""
          },
          {
            "text": "Cobalt2",
            "detail": "dark",
            "keys": ""
          },
          {
            "text": "Dracula",
            "detail": "dark",
            "keys": ""
          },
          {
            "text": "GitHub",
            "detail": "",
            "keys": ""
          },
          {
            "text": "Gruvbox",
            "detail": "",
            "keys": ""
          },
          {
            "text": "Monokai",
            "detail": "dark",
            "keys": ""
          },
          {
            "text": "Nord",
            "detail": "dark",
            "keys": ""
          },
          {
            "text": "One Dark",
            "detail": "dark",
            "keys": ""
          },
          {
            "text": "Solarized",
            "detail": "",
            "keys": ""
          },
          {
            "text": "Tokyo Night",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.tools.index.stats",
    "name": "Index stats",
    "version": "2026.11.12",
    "description": "Shows in the title bar how many vectors the search index holds",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Slots",
        "items": [
          {
            "text": "titleBar.status",
            "detail": "stats",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "index:read",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.tools.index.stats",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.tools.media.importer",
    "name": "Media Importer",
    "version": "2026.11.18",
    "description": "Saves web articles and downloads online videos for offline playback",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [
      "tools"
    ],
    "settings": [
      {
        "key": "ytDlp",
        "type": "string",
        "title": "yt-dlp",
        "description": "Path to the yt-dlp program",
        "default": ""
      },
      {
        "key": "ffmpeg",
        "type": "string",
        "title": "ffmpeg",
        "description": "Path to the ffmpeg program",
        "default": ""
      },
      {
        "key": "captions",
        "type": "string",
        "title": "Caption languages",
        "description": "Preferred caption languages, comma-separated",
        "default": ""
      },
      {
        "key": "automaticCaptions",
        "type": "boolean",
        "title": "Automatic captions",
        "description": "Use auto-generated captions when a site has no others",
        "default": true
      },
      {
        "key": "downloadCaptionsAutomatically",
        "type": "boolean",
        "title": "Download captions automatically",
        "description": "Download captions or the article of every new link note",
        "default": false
      },
      {
        "key": "copyMaxSizeMb",
        "type": "number",
        "title": "Maximum download size",
        "description": "Largest video to download, in MB; 0 for no limit",
        "default": 500,
        "min": 0,
        "max": 100000,
        "step": 50
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Commands",
        "items": [
          {
            "text": "Import link",
            "detail": "link.import",
            "keys": ""
          },
          {
            "text": "Download media",
            "detail": "link.downloadMedia",
            "keys": ""
          },
          {
            "text": "Delete media",
            "detail": "link.deleteMedia",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Handlers",
        "items": [
          {
            "text": "Download captions",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Triggers",
        "items": [
          {
            "text": "file.added",
            "detail": "app:file.createText",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Text kinds",
        "items": [
          {
            "text": "Article",
            "detail": "article",
            "keys": ""
          },
          {
            "text": "Captions",
            "detail": "captions",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Note types",
        "items": [
          {
            "text": "link",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "fileTree.context",
            "detail": "app:file.createText",
            "keys": ""
          },
          {
            "text": "fileTree.context",
            "detail": "link.downloadMedia",
            "keys": ""
          },
          {
            "text": "fileTree.context",
            "detail": "link.deleteMedia",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Drop handlers",
        "items": [
          {
            "text": "text/uri-list",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Frame origins",
        "items": [
          {
            "text": "https://www.youtube-nocookie.com",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "link:open",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.tools.media.importer",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Capabilities",
        "items": [
          {
            "text": "tools",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.tools.proofreading",
    "name": "LLM Proofreading",
    "version": "2026.11.11",
    "description": "Cleans up transcripts and scanned text with a language model",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "connections",
        "type": "connections",
        "title": "Connections",
        "description": "Services and agents, tried in order until one answers",
        "max": 5,
        "operations": [
          "llm.complete"
        ]
      },
      {
        "key": "batchSize",
        "type": "number",
        "title": "Batch size",
        "description": "Lines per model request",
        "default": 40,
        "min": 1,
        "max": 500,
        "step": 1
      },
      {
        "key": "overlap",
        "type": "number",
        "title": "Overlap",
        "description": "Lines shared by neighboring batches",
        "default": 4,
        "min": 0,
        "max": 100,
        "step": 1
      },
      {
        "key": "inFlight",
        "type": "number",
        "title": "In-flight requests",
        "description": "Parallel model requests per call",
        "default": 4,
        "min": 1,
        "max": 32,
        "step": 1
      },
      {
        "key": "maxEditDistance",
        "type": "number",
        "title": "Max edit distance",
        "description": "Largest change a correction may make to a line",
        "default": 0.3,
        "min": 0,
        "max": 1,
        "step": 0.05
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Commands",
        "items": [
          {
            "text": "Proofread text",
            "detail": "proofreading.create",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Text kinds",
        "items": [
          {
            "text": "Proofread text",
            "detail": "proofreading",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.tools.search.relevance",
    "name": "Search Relevance",
    "version": "2026.11.8",
    "description": "Sets how similar a note must be to your search to appear in the results",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [
      {
        "id": "floor",
        "title": "Search relevance"
      }
    ],
    "contributions": [
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "index:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "search:policy",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.tools.search.relevance",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Activation events",
        "items": [
          {
            "text": "onSearchSourceReady",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.tools.updater",
    "name": "Update Notifier",
    "version": "2026.11.10",
    "description": "Notifies when a newer build of Numen is published",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "channel",
        "type": "enum",
        "title": "Channel",
        "description": "Which builds are announced",
        "default": "stable",
        "options": [
          {
            "value": "stable",
            "title": "Stable",
            "detail": "Tested releases"
          },
          {
            "value": "beta",
            "title": "Beta",
            "detail": "Builds published ahead of the next release"
          },
          {
            "value": "custom",
            "title": "Custom",
            "detail": "Builds published at the release feed address"
          }
        ]
      },
      {
        "key": "endpoint",
        "type": "string",
        "title": "Release feed address",
        "description": "Address of latest.json for the Custom channel",
        "default": "",
        "when": {
          "setting": "channel",
          "is": [
            "custom"
          ]
        }
      },
      {
        "key": "checkIntervalHours",
        "type": "number",
        "title": "Check interval",
        "description": "Hours between two update checks",
        "default": 4,
        "min": 1,
        "max": 168,
        "step": 1
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Slots",
        "items": [
          {
            "text": "titleBar.status",
            "detail": "update",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "link:open",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.tools.updater",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.agent.chat",
    "name": "Agent Chat",
    "version": "2026.11.12",
    "description": "Interactive chat tab for working with AI assistants",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "connections",
        "type": "connections",
        "title": "Agents",
        "description": "",
        "max": 5,
        "operations": [
          "acp.task"
        ]
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Commands",
        "items": [
          {
            "text": "Open agent chat",
            "detail": "chat.open",
            "keys": "mod+shift+a"
          },
          {
            "text": "Ask agent about note",
            "detail": "note.ask",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "Agent",
            "detail": "agent",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "titleBar.commands",
            "detail": "chat.open",
            "keys": ""
          },
          {
            "text": "fileTree.context",
            "detail": "note.ask",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "note.ask",
            "keys": ""
          }
        ]
      },
      {
        "kind": "View capabilities",
        "items": [
          {
            "text": "conversation",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.file.tree",
    "name": "File Tree Explorer",
    "version": "2026.11.16",
    "description": "Sidebar file browser for managing notes, books, and recordings",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Commands",
        "items": [
          {
            "text": "Open file tree",
            "detail": "tree.open",
            "keys": "mod+shift+e"
          },
          {
            "text": "Rename",
            "detail": "tree.renameFile",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Keybindings",
        "items": [
          {
            "text": "app:file.reveal",
            "detail": "",
            "keys": "mod+alt+e"
          }
        ]
      },
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "Files",
            "detail": "files",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "palette.searchResult",
            "detail": "app:file.reveal",
            "keys": ""
          },
          {
            "text": "palette.note",
            "detail": "app:file.reveal",
            "keys": ""
          },
          {
            "text": "titleBar.commands",
            "detail": "tree.open",
            "keys": ""
          },
          {
            "text": "fileTree.context",
            "detail": "fileTree.new",
            "keys": ""
          },
          {
            "text": "fileTree.new",
            "detail": "app:note.create",
            "keys": ""
          },
          {
            "text": "fileTree.new",
            "detail": "app:folder.create",
            "keys": ""
          },
          {
            "text": "fileTree.context",
            "detail": "tree.renameFile",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:file.reveal",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.flashcards.studio",
    "name": "Flashcards Studio",
    "version": "2026.11.19",
    "description": "Flashcard deck editor, card template designer, and spaced-repetition review",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [
      "tools"
    ],
    "settings": [
      {
        "key": "dayStartTime",
        "type": "string",
        "title": "Day starts at",
        "description": "Time a review day starts, HH:MM up to 12:00",
        "default": "04:00"
      },
      {
        "key": "agentConnections",
        "type": "connections",
        "title": "Agents",
        "description": "",
        "max": 5,
        "operations": [
          "acp.task"
        ]
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Commands",
        "items": [
          {
            "text": "Create deck",
            "detail": "deck.create",
            "keys": ""
          },
          {
            "text": "Create card model",
            "detail": "cardModel.create",
            "keys": ""
          },
          {
            "text": "Create preset",
            "detail": "preset.create",
            "keys": ""
          },
          {
            "text": "Open preset",
            "detail": "preset.open",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "Deck",
            "detail": "deck",
            "keys": ""
          },
          {
            "text": "Card Model",
            "detail": "card-model",
            "keys": ""
          },
          {
            "text": "Preset",
            "detail": "preset",
            "keys": ""
          },
          {
            "text": "Flashcards",
            "detail": "flashcards.review",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Note types",
        "items": [
          {
            "text": "deck",
            "detail": "",
            "keys": ""
          },
          {
            "text": "card-model",
            "detail": "",
            "keys": ""
          },
          {
            "text": "preset",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "fileTree.new",
            "detail": "deck.create",
            "keys": ""
          },
          {
            "text": "fileTree.new",
            "detail": "cardModel.create",
            "keys": ""
          },
          {
            "text": "fileTree.new",
            "detail": "preset.create",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "preset.open",
            "keys": ""
          }
        ]
      },
      {
        "kind": "View capabilities",
        "items": [
          {
            "text": "vault-summary",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Services",
        "items": [
          {
            "text": "md.numen.views.flashcards.studio",
            "detail": "",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Capabilities",
        "items": [
          {
            "text": "tools",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.media.viewer",
    "name": "Media Viewer & Recorder",
    "version": "2026.11.2",
    "description": "Audio and video player with interactive transcripts and voice recording",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Commands",
        "items": [
          {
            "text": "Record audio",
            "detail": "recording.create",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "Record Audio",
            "detail": "recording",
            "keys": ""
          },
          {
            "text": "Audio Player",
            "detail": "reader:media-viewer",
            "keys": ""
          },
          {
            "text": "Media",
            "detail": "media",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Readers",
        "items": [
          {
            "text": "media-viewer",
            "detail": "audio/mpeg, audio/mp4, audio/vnd.wave, audio/ogg, audio/aac, audio/flac, video/mp4, video/webm",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.plex.graph",
    "name": "Plex Graph",
    "version": "2026.11.19",
    "description": "Interactive visual map of connected notes and topics",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "nestHeadings",
        "type": "boolean",
        "title": "Nest headings under notes",
        "description": "Show the headings of a note under its node",
        "default": true
      },
      {
        "key": "headingLimit",
        "type": "number",
        "title": "Headings per note",
        "description": "Most headings shown under a node",
        "default": 6,
        "min": 1,
        "max": 12,
        "step": 1
      },
      {
        "key": "showPills",
        "type": "boolean",
        "title": "Pill background",
        "description": "Draw a pill behind each title",
        "default": true
      },
      {
        "key": "pillPaddingSides",
        "type": "number",
        "title": "Pill padding, sides",
        "description": "Space beside the title inside its pill",
        "default": 10,
        "min": 0,
        "max": 24,
        "step": 1
      },
      {
        "key": "pillPaddingEnds",
        "type": "number",
        "title": "Pill padding, top and bottom",
        "description": "Space above and below the title inside its pill",
        "default": 9.25,
        "min": 0,
        "max": 16,
        "step": 0.25
      },
      {
        "key": "focusSize",
        "type": "number",
        "title": "Focus size",
        "description": "How much larger the pill of the middle note is",
        "default": 1.22,
        "min": 1,
        "max": 2.5,
        "step": 0.01
      },
      {
        "key": "animationLength",
        "type": "number",
        "title": "Animation",
        "description": "Time to move to another note, in ms; 0 is instant",
        "default": 420,
        "min": 0,
        "max": 2000,
        "step": 20
      },
      {
        "key": "lineThickness",
        "type": "number",
        "title": "Line thickness",
        "description": "Width of the lines between notes, in pixels.",
        "default": 2,
        "min": 0.5,
        "max": 4,
        "step": 0.25
      },
      {
        "key": "outlineThickness",
        "type": "number",
        "title": "Outline thickness",
        "description": "Width of the outline of a pill, in pixels.",
        "default": 1,
        "min": 0.5,
        "max": 4,
        "step": 0.25
      },
      {
        "key": "lineCurvature",
        "type": "number",
        "title": "Line curvature",
        "description": "How far lines bend; 0 is straight",
        "default": 0.55,
        "min": 0,
        "max": 1,
        "step": 0.05
      },
      {
        "key": "noteRoom",
        "type": "number",
        "title": "Room around a note",
        "description": "Empty space around every note",
        "default": 0,
        "min": 0,
        "max": 40,
        "step": 1
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Commands",
        "items": [
          {
            "text": "Open plex graph",
            "detail": "graph.open",
            "keys": "mod+shift+x"
          }
        ]
      },
      {
        "kind": "Keybindings",
        "items": [
          {
            "text": "app:file.reveal",
            "detail": "",
            "keys": "mod+shift+p"
          }
        ]
      },
      {
        "kind": "Tab kinds",
        "items": [
          {
            "text": "Plex",
            "detail": "plex",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Slots",
        "items": [
          {
            "text": "palette.searchResult",
            "detail": "app:file.reveal",
            "keys": ""
          },
          {
            "text": "palette.note",
            "detail": "app:file.reveal",
            "keys": ""
          },
          {
            "text": "titleBar.commands",
            "detail": "graph.open",
            "keys": ""
          },
          {
            "text": "fileTree.context",
            "detail": "app:file.reveal",
            "keys": ""
          },
          {
            "text": "fileTree.context",
            "detail": "plex.relation",
            "keys": ""
          },
          {
            "text": "plex.relation",
            "detail": "app:note.createChild",
            "keys": ""
          },
          {
            "text": "plex.relation",
            "detail": "app:note.createParent",
            "keys": ""
          },
          {
            "text": "plex.relation",
            "detail": "app:note.createJump",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:note.create",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:file.open",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:file.reveal",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:file.copyPath",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:note.createChild",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:note.createParent",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:note.createJump",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:note.rename",
            "keys": ""
          },
          {
            "text": "plexGraph.context",
            "detail": "app:file.remove",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.statusBar",
    "name": "Status bar",
    "version": "2026.11.4",
    "description": "The bar at the bottom of the window with plugin commands",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Slots",
        "items": [
          {
            "text": "statusBar",
            "detail": "bar",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.tasks",
    "name": "Tasks",
    "version": "2026.11.5",
    "description": "Background tasks of the window, shown in the title bar",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Slots",
        "items": [
          {
            "text": "titleBar.status",
            "detail": "status",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  },
  {
    "id": "md.numen.views.titleBar",
    "name": "Title bar",
    "version": "2026.11.4",
    "description": "Draws the bar at the top of each window: the commands, the vault and the window buttons",
    "isActive": true,
    "hasFrontend": true,
    "hasBackend": true,
    "capabilities": [],
    "settings": [
      {
        "key": "buttons",
        "type": "enum",
        "title": "Window buttons",
        "description": "Where the bar draws the window buttons",
        "default": "windows",
        "options": [
          {
            "value": "mac",
            "title": "Left",
            "detail": "Round buttons at the left, as on macOS."
          },
          {
            "value": "windows",
            "title": "Right",
            "detail": "Square buttons at the right, as on Windows."
          },
          {
            "value": "none",
            "title": "None",
            "detail": "The bar draws no window buttons."
          }
        ]
      },
      {
        "key": "hideInFullscreen",
        "type": "boolean",
        "title": "Hide in full screen",
        "description": "Hide the bar while the window is full screen",
        "default": false
      }
    ],
    "settingsViews": [],
    "contributions": [
      {
        "kind": "Slots",
        "items": [
          {
            "text": "titleBar",
            "detail": "bar",
            "keys": ""
          }
        ]
      },
      {
        "kind": "Permissions",
        "items": [
          {
            "text": "vault:read",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:write",
            "detail": "",
            "keys": ""
          },
          {
            "text": "vault:manage",
            "detail": "",
            "keys": ""
          }
        ]
      }
    ]
  }
] as unknown as readonly PluginInfo[]

export const SHIPPED_CATALOG: readonly CatalogEntry[] = [
  {
    "id": "md.numen.agents.acp",
    "name": "ACP Agents",
    "version": "2026.11.14",
    "description": "External AI assistants via the Agent Client Protocol: Claude Code, Codex, Antigravity and any other ACP program",
    "category": "agents",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.agents.mcp.server",
    "name": "Model Context Protocol Server",
    "version": "2026.11.15",
    "description": "Connects external AI apps to your open vault",
    "category": "agents",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.agents.numen",
    "name": "Numen Agent",
    "version": "2026.11.12",
    "description": "Built-in AI assistant that chats through a connection to OpenRouter, OpenAI or an OpenAI-compatible server",
    "category": "agents",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.agents.tools.vault",
    "name": "Vault Knowledge Tools",
    "version": "2026.11.8",
    "description": "Library search, reading, and note editing tools for AI assistants",
    "category": "agents",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.agents.tools.window",
    "name": "Window Interaction Tools",
    "version": "2026.11.8",
    "description": "Workspace context and interactive edit previews for AI assistants",
    "category": "agents",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.connections.llm",
    "name": "Language model services",
    "version": "2026.11.2",
    "description": "Connects Numen to OpenRouter, OpenAI and OpenAI-compatible servers for completions, chat and embeddings",
    "category": "connections",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.engines.embeddings",
    "name": "Vector Embeddings",
    "version": "2026.11.13",
    "description": "Meaning-based search across your notes, books, and recordings",
    "category": "engines",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.engines.ocr",
    "name": "Document OCR",
    "version": "2026.11.19",
    "description": "Text recognition for scanned PDFs and images",
    "category": "engines",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.engines.transcription",
    "name": "Audio/Video Transcription",
    "version": "2026.11.19",
    "description": "Offline speech-to-text transcription for audio and video recordings",
    "category": "engines",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.notes",
    "name": "Notes",
    "version": "2026.11.9",
    "description": "Markdown note editor with live formatting and wikilinks",
    "category": "views",
    "feature": "Markdown notes",
    "tabKinds": [
      "note",
      "reader:markdown"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.readers.epub",
    "name": "EPUB Book Reader",
    "version": "2026.11.16",
    "description": "EPUB book reader with highlights and passage links",
    "category": "readers",
    "feature": "EPUB books",
    "tabKinds": [
      "reader:reader-epub"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.readers.pdf",
    "name": "PDF Reader",
    "version": "2026.11.13",
    "description": "PDF document reader with highlights and passage links",
    "category": "readers",
    "feature": "PDF documents",
    "tabKinds": [
      "reader:reader-pdf"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.themes.presets",
    "name": "Preset Themes",
    "version": "2026.11.2",
    "description": "Twelve color themes for the editor and the flashcards window",
    "category": "themes",
    "feature": "color themes",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.tools.index.stats",
    "name": "Index stats",
    "version": "2026.11.12",
    "description": "Shows in the title bar how many vectors the search index holds",
    "category": "tools",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.tools.media.importer",
    "name": "Media Importer",
    "version": "2026.11.18",
    "description": "Saves web articles and downloads online videos for offline playback",
    "category": "tools",
    "feature": "media links",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.tools.proofreading",
    "name": "LLM Proofreading",
    "version": "2026.11.11",
    "description": "Cleans up transcripts and scanned text with a language model",
    "category": "tools",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.tools.search.relevance",
    "name": "Search Relevance",
    "version": "2026.11.8",
    "description": "Sets how similar a note must be to your search to appear in the results",
    "category": "tools",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.tools.updater",
    "name": "Update Notifier",
    "version": "2026.11.10",
    "description": "Notifies when a newer build of Numen is published",
    "category": "tools",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.agent.chat",
    "name": "Agent Chat",
    "version": "2026.11.12",
    "description": "Interactive chat tab for working with AI assistants",
    "category": "views",
    "feature": "agent chat",
    "tabKinds": [
      "agent"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.file.tree",
    "name": "File Tree Explorer",
    "version": "2026.11.16",
    "description": "Sidebar file browser for managing notes, books, and recordings",
    "category": "views",
    "feature": "file tree",
    "tabKinds": [
      "files"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.flashcards.studio",
    "name": "Flashcards Studio",
    "version": "2026.11.19",
    "description": "Flashcard deck editor, card template designer, and spaced-repetition review",
    "category": "views",
    "feature": "flashcards",
    "tabKinds": [
      "deck",
      "card-model",
      "preset",
      "flashcards.review"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.media.viewer",
    "name": "Media Viewer & Recorder",
    "version": "2026.11.2",
    "description": "Audio and video player with interactive transcripts and voice recording",
    "category": "views",
    "feature": "audio and video playback and recording",
    "tabKinds": [
      "recording",
      "reader:media-viewer",
      "media"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.plex.graph",
    "name": "Plex Graph",
    "version": "2026.11.19",
    "description": "Interactive visual map of connected notes and topics",
    "category": "views",
    "feature": "plex graph",
    "tabKinds": [
      "plex"
    ],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.statusBar",
    "name": "Status bar",
    "version": "2026.11.4",
    "description": "The bar at the bottom of the window with plugin commands",
    "category": "views",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.tasks",
    "name": "Tasks",
    "version": "2026.11.5",
    "description": "Background tasks of the window, shown in the title bar",
    "category": "views",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  },
  {
    "id": "md.numen.views.titleBar",
    "name": "Title bar",
    "version": "2026.11.4",
    "description": "Draws the bar at the top of each window: the commands, the vault and the window buttons",
    "category": "views",
    "feature": "",
    "tabKinds": [],
    "isAvailable": false
  }
]

export const SHIPPED_READMES: Readonly<Record<string, string>> = {
  "md.numen.agents.acp": "Use the AI assistants installed on your computer, such as Claude Code, Codex and Antigravity, in the Agent Chat panel and for AI tasks across your vault.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Chat with your favorite AI tools**: Add a connection of the type Claude Code, OpenAI Codex, Antigravity or ACP agent, and choose it as an agent of the chat. The assistant answers in the chat as it writes.\n- **Work directly with your vault**: The assistant can call the tools of your vault, such as searching your notes and reading passages from books and documents. It may call these tools and no others: it reads no file of your computer, runs no command, and of its own tools Claude Code keeps web search alone.\n- **Continue a conversation**: The turns of one chat continue one session of the assistant. A session unused for 30 minutes is closed, and at most 8 are kept.\n- **Any ACP assistant**: The type ACP agent runs any program that speaks the Agent Client Protocol, such as `gemini --acp`, `opencode acp`, `cursor-agent acp`, `goose acp`, `kimi acp` and `qwen --acp`. Choose the program of the connection.\n- **Answer requests of other plugins**: A request of another plugin runs in an empty temporary folder, with no tools and without writing to disk.\n\n## Programs\n\nClaude Code, Codex and Antigravity speak ACP through an adapter. Leave the program of the connection empty to use the one on your PATH:\n\n- Claude Code: `claude-agent-acp` (npm `@agentclientprotocol/claude-agent-acp`) or `claude-code-acp`.\n- Codex: `codex-acp`.\n- Antigravity: `agy`, started as `agy acp`.\n\n## Settings\n\n- **Request timeout**: Seconds to wait for the program to answer a request from a connection, or to open the session of a task. Without a value, it waits 120 seconds.\n",
  "md.numen.agents.mcp.server": "Let external AI apps—such as Claude Desktop, Cursor, or Windsurf—read, search, and work with your open Numen vault.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Use your vault from other AI apps**: Ask an external AI assistant to search your notes, read passages from your books, or create new notes while you work in another app.\n- **Same vault**: Changes and searches apply to the vault that is open in Numen.\n\n## How to Connect\n\nWhile the plugin is enabled, Numen listens on `127.0.0.1` at the plugin's **Port** setting (default `3333`), listens on the new port when the setting changes, and starts the MCP server when a client connects:\n\n- **Streamable HTTP endpoint**: `http://127.0.0.1:3333/mcp`\n\nA client authenticates with the header `Authorization: Bearer <token>`, where `<token>` is the value of the plugin's **Token** setting on its page in the settings. Numen generates the token, 64 hexadecimal characters, when it first loads the enabled plugin; copy it from the setting into the client configuration. You can replace it with a token of your own, and while the setting is empty the server returns 401 for every request. A changed token takes effect at once. A request sent from a web page whose origin is not `localhost` or a loopback address gets 403.\n\n### Example JSON Configuration (`mcpServers`)\n\nAdd the following entry to your client's MCP configuration file (such as `claude_desktop_config.json`, `.cursor/mcp.json`, or `~/.codeium/windsurf/mcp_config.json`), with your token in place of `<token>`:\n\n```json\n{\n  \"mcpServers\": {\n    \"numen\": {\n      \"type\": \"http\",\n      \"url\": \"http://127.0.0.1:3333/mcp\",\n      \"headers\": { \"Authorization\": \"Bearer <token>\" }\n    }\n  }\n}\n```\n",
  "md.numen.agents.numen": "Built-in AI assistant that chats through a connection to a cloud service (OpenRouter, OpenAI) or to a server on your computer (Ollama, LM Studio).\n\n<!-- more -->\n\n## What You Can Do\n\n- **Chat and research immediately**: Add a connection in Settings, choose it in the Connection setting of this plugin, add a connection of the type Numen agent, choose it in the Agents setting of the agent chat, and start chatting with your vault.\n- **Pick the model**: The row of the setting holds the connection, the model and the reasoning level. The models of the connection are listed, and you can type any model name yourself.\n- **Full vault access**: Can search your notes and books, follow links, and suggest edits in your workspace, with the tools of the other plugins that its permissions allow.\n\n## Settings\n\n- **Parallel requests** (on the Numen agent connection): How many tasks of the connection run at once (1 by default).\n- **Connection**: One row with the connection to chat through, the model and the reasoning level. Only connections that support chat are offered. The address and the key belong to the connection.\n- **Temperature**: How creative or focused the replies should be (`0.0` is more focused, `1.0` is more creative).\n- **Max tokens**: The maximum length of each reply.\n- **Max steps**: How many requests to the model one task may make before it is stopped (25 by default).\n",
  "md.numen.agents.tools.vault": "Allows AI assistants in Numen to search your library, read notes and books, create or update notes, and explore links between ideas.\n\n<!-- more -->\n\n## What It Enables\n\n- **Smart library search**: Lets the assistant find relevant passages across your notes, books, PDFs, and transcripts—both by exact words and by meaning.\n- **Reading and quoting sources**: Allows the assistant to open specific pages or timestamps so its answers include clickable links back to the original passage.\n- **Creating and organizing notes**: Lets the assistant draft new notes, update existing ones, and connect related notes with links.\n- **Viewing the link graph**: Helps the assistant see which notes link to each other and discover isolated or unlinked notes.\n- **Managing notes and vaults**: Lets the assistant rename and move notes, move them to the vault's trash, change the links in their frontmatter, and add, rename, remove and open vaults.\n",
  "md.numen.agents.tools.window": "Allows AI assistants to see which tab and passage you are currently reading and to show you interactive previews before changing a note.\n\n<!-- more -->\n\n## What It Enables\n\n- **Context awareness**: When you ask a question like *\"Summarize this page\"*, the assistant can see which note, book, PDF, or recording you have open.\n- **Guided navigation**: Lets the assistant open a note, book passage, or transcript right in your workspace when you ask for it.\n- **Safe edit previews**: Instead of changing your notes silently, the assistant highlights proposed additions and removals right inside the editor so you can accept or reject each change.\n",
  "md.numen.connections.llm": "Connects Numen to OpenRouter, OpenAI and servers that speak the OpenAI API, such as Ollama or LM Studio, so that plugins can use their models for completions and embeddings.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Add a service as a connection**: Pick OpenRouter, OpenAI or an OpenAI-compatible server when you add a connection. The key is kept in the system keychain and is never shown to the plugins that use the connection.\n- **Use your own server**: For an OpenAI-compatible server, enter its address, for example `http://localhost:11434/v1`. A key is optional. A key is sent over `http` only to this computer, and over `https` to any address.\n- **Choose a model for each use**: The models the service lists are offered where you choose a model. You can also type a model name the service does not list.\n- **Set reasoning**: OpenRouter and the reasoning models of OpenAI take a level of low, medium or high.\n\n## Settings\n\n- **Request timeout**: How many seconds to wait for a service to answer. Without a value a completion waits 120 seconds and an embedding 60 seconds. The first request to a model of an OpenAI-compatible server waits up to 10 minutes, because a local server loads the model on it.\n",
  "md.numen.engines.embeddings": "Provides meaning-based search across your vault so you can find related notes, books, and recordings even when they use different words.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Search by meaning, not just keywords**: Find passages that discuss an idea or topic even if they don't contain the exact words you typed.\n- **Run privately on your computer or through a connection**: Use the built-in local model offline on your machine, or use a connection to OpenRouter, OpenAI, or another service for fast indexing of large libraries.\n\n## Settings\n\n- **Source**:\n  - **Local model**: Runs privately on your computer at no cost and supports over 100 languages. The performance profile of the settings sizes its threads.\n  - **Connection**: Sends the texts to a service through a connection you add on the Connections page of the settings. The plugin has no key and no address of its own.\n- **Model** (Local model): Pick one of the suggested models from the list: E5 small (the default, 118 MB, runs well on a CPU), Harrier 270M (better results, 344 MB) and BGE-M3 (large, 569 MB). Any other local model needs a profile of your own, described below.\n- **Connection** (Connection): One row with a connection and an embedding model. Choosing another connection or model indexes the vault again, and the editor asks first because the service may charge for it. Editing the connection so that its address or the profile of the model changes indexes the vault again as well. If the service rejects the key or asks for payment, indexing stops and the status bar names the connection and the reason.\n- **Smallest piece of text to index for meaning**: Pieces of text shorter than this many characters, such as page numbers and running heads, get no vector, so they do not crowd out real answers. They stay searchable by their words. A note shorter than this is found by its words too. The default is 40, and 0 gives every piece a vector. Changing it does not make the other vectors stale: no text is embedded again, and the vectors of the pieces below the new value are removed.\n- **Longest chunk of a note, in tokens**: A note is cut into chunks of at most this many tokens, and each chunk gets a vector. The chunk never exceeds the input limit of the model. A smaller value gives more precise passages and more vectors. The default is 256, and the value is held between 16 and 8192. A change cuts the notes again in every open vault; the vectors of texts that did not move are kept.\n\n## Model profiles\n\nA local model is run as its profile says: the profile is data, not code. The profiles that ship with the plugin are in `backend/domain/model/profiles.json`. To add a model or change one without rebuilding, put a `*.json` file in the `profiles` folder of the plugin data directory (the folder the host grants the plugin as `NUMEN_PLUGIN_DATA_DIR`). A file holds one profile or an array of them, and a profile is merged over the shipped profile with the same `id`, field by field, so a file that gives only `queryPrefix` changes that and nothing else. A file that is not valid JSON, names a field the plugin does not know, or holds a profile that cannot run is reported on the log of the plugin and skipped whole. The files are read again each time a setting of the plugin changes.\n\nA profile has these fields. `id` is the Hugging Face repository of a local model, and the value of the Model setting. `title`, `tier` and `sizeMB` describe it in the list. `dims` is the width of the vectors, and `maxTokens` the longest text the model takes; the plugin gives a local model at most 512 tokens whatever this says. `pooling` is how the output becomes a vector: `mean` of the tokens, `cls` (the first token), `last` (the last token), or `output` (the graph answers one vector per text as `sentence_embedding`). `normalize` scales the vector to unit length and is true when absent. `queryPrefix` and `documentPrefix` are put in front of a text that is searched with and a text that is searched, and both may be empty. `onnxFile` is the model file in the repository, such as `onnx/model_quantized.onnx`; a file with a `_data` file beside it is fetched with it. `revision` and `sha256` pin the repository to a commit and each file to its sha256, with the path in the repository as the key, and are optional; a profile that changes `onnxFile` and gives no `sha256` drops the shipped pins. `extraInputs` lists inputs besides the tokens and the mask, and only `position_ids` is known. `floorHint` is the starting similarity a passage has to reach to answer a query, and 0 means unknown. A model with no profile is refused with an error that names the folder, and a model whose graph asks for an input the profile does not list is refused. The variables `NUMEN_EMBEDDING_DIMENSIONS`, `NUMEN_EMBEDDING_MAX_TOKENS`, `NUMEN_EMBEDDING_POOLING`, `NUMEN_EMBEDDING_REPO`, `NUMEN_EMBEDDING_DIR` and `NUMEN_EMBEDDING_FILE` still override the profile that is chosen.\n\nA text is embedded for one of two roles. The index embeds the pieces of your notes as documents, and a question you type is embedded as a query. A model that tells the two apart, such as E5, reads each with the prefix its profile names. A connection is given the prefixes of the profile that its provider reports for the model, which are empty for a model the provider has no profile for.\n\n### After a change of the profile\n\nThe vectors are stored under a recipe that holds a short hash of the prefixes, the model file, the token limit and the normalization. For a connection the recipe takes the type of the connection with its address as the source, and the name, width, input limit, pooling and prefixes from the profile that the provider gives before any text is embedded, so the index is fitted before the first batch. The plugin asks for the profile once for each configuration, and the host sends a configuration again when the connection is edited. Any change of those makes the vectors of the old profile stale: the plugin embeds the vault again under the new recipe in the background, and removes the old vectors when it is done. The smallest piece of text to index is not part of the hash. The default model now reads documents and queries with the E5 prefixes, which it did not before, so a vault embedded earlier is embedded once more; this takes about one to two and a half hours for a large vault with the performance profile, and search by words works meanwhile.\n\n## Search Relevance\n\nThis plugin tells Numen which model it runs and a starting value for the model's threshold, the similarity a note has to reach to show up in a search. Choosing the threshold, measuring it on your vault and checking a query against it is the job of the **Search Relevance** plugin.\n",
  "md.numen.engines.ocr": "Recognizes text in scanned PDFs and images on your computer so their contents can be searched, read by the agent, and highlighted when opened from search results or links.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Make scans searchable**: Run **Recognize text** from a PDF or image file's right-click menu in the file tree, or turn on **Recognize automatically** to recognize new files when they are added to the vault.\n- **Open matched passages on scanned pages**: Stores each recognized line with its page number and bounding box in a `recognition` text version, so search results and agent links open a scanned PDF at that page and highlight the passage on the page.\n- **Resume after closing**: Saves progress every eight pages while a document is processed, so an interrupted run continues from the last saved batch.\n- **Ask the agent to recognize a scan**: The agent in the chat panel can start the recognition of a scan with the tool `recognition_create` and read the text once it is recognized.\n\n## Settings\n\n- **Model**: Choose the recognition model that matches your documents (`PP-OCRv6 tiny`, `PP-OCRv6 small`, `PP-OCRv5 Latin`, `PP-OCRv5 East Slavic`, `PP-OCRv5 Cyrillic`, `PP-OCRv5 Devanagari`, or `PP-OCRv5 Greek`), or choose **Custom model** to use your own ONNX **Model file** and **Character dictionary**.\n- **Scan resolution**: Resolution a page is rendered at before recognition (`150 dpi`, `200 dpi`, `300 dpi` by default, `400 dpi`, or `600 dpi`).\n- **Recognize automatically**: Recognize the text of every PDF and image in the vault that has none (`false` by default).\n",
  "md.numen.engines.transcription": "Turns your audio and video recordings into interactive, time-stamped transcripts right on your computer.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Transcribe lectures, interviews, and voice notes**: Click **Transcribe** inside the Media Viewer or right-click any audio or video file in the sidebar to generate a full transcript offline.\n- **Read along as you listen**: Each line of the transcript is linked to its timestamp—click any line to jump the recording to that moment, and search your recordings just like text notes.\n- **Ask the agent to transcribe a recording**: The agent in the chat panel can start the transcription of a recording with the tool `transcript_create` and read the transcript as it is written.\n\n## Settings\n\n- **Model**:\n  - **Parakeet TDT 0.6B v3** _(default)_: Supports 25 European languages and detects the spoken language automatically.\n  - **Parakeet TDT 0.6B v2**: Optimized specifically for English recordings.\n",
  "md.numen.notes": "Write and organize Markdown notes with live formatting, `[[wikilinks]]`, tables, and instant links to passages in your books and recordings.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Clean live writing**: Headings, lists, tables, quotes, and links are formatted cleanly as you read, while showing the underlying Markdown syntax on the line you are currently editing.\n- **Connect your ideas**: Link notes together with `[[Note Title]]` and paste quotes from books, PDFs, or transcripts that open the exact page or timestamp when clicked.\n- **Interactive AI review**: When an AI assistant suggests changes to a note, review and accept or reject each edit right inside the note.\n- **Create a note from the title bar**: The title bar has a New note button, placed by this plugin, that asks for the name of the note in the command palette.\n\n## Settings\n\n- **Live Preview**: Format Markdown cleanly as you type, or turn off to view raw Markdown everywhere.\n- **Line Numbers**: Show or hide line numbers along the left margin of your notes.\n- **Soft Wrap**: Wrap long lines to fit the window width, or scroll horizontally.\n",
  "md.numen.readers.epub": "Opens EPUB books in a two-column or one-column page reader, extracts their text for search and the agent, and highlights passages opened from links and search results.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Read EPUB books page by page**: Opens `.epub` files in a reader tab laid out in one or two columns depending on the pane width. Turn pages with the arrow keys, Page Up and Page Down, the space bar, a drag, or a click near either edge.\n- **Browse and filter the table of contents**: Open the contents list from the button in the bottom-left corner, filter sections by title, and jump to any chapter or section.\n- **Follow links and keep your place**: Click internal book links to move between sections, and reopen a book at the position where you left off.\n- **Index book text for search and the agent**: Automatically extracts the text of each EPUB added to the vault into a `book-text` version, or run **Extract text** from a file's right-click menu in the file tree.\n- **Open passages from search and links**: Opening a search result, a `numen:` passage link, or an agent citation jumps to that place in the book and highlights the passage on the page.\n",
  "md.numen.readers.pdf": "Opens PDF documents in a horizontal page reader with zoom controls, extracts their text layer for search and the agent, and highlights passages opened from links and search results.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Read PDF pages**: Opens `.pdf` files in a horizontal strip of pages. Move between pages by dragging, scrolling, pressing the arrow keys, Page Up, Page Down, Home, or End, clicking the previous and next page buttons, or typing a page number in the toolbar.\n- **Zoom pages**: Zoom in and out from the toolbar while keeping the current view centered, and reopen a PDF at the page where you left off.\n- **Extract the text layer**: Automatically extracts the text layer and outline headings of each PDF added to the vault into a `page-text` version, or run **Extract text layer** from a file's right-click menu in the file tree.\n- **Open passages from search and links**: Opening a search result, a `numen:` passage link, or an agent citation jumps to that page and highlights the matched passage boxes, using either the PDF's own text layer or text recognized by **Document OCR**.\n\n## Settings\n\n- **Default Zoom**: Initial zoom level when opening a PDF document (`page`, `width`, or `100%`; `page` by default).\n",
  "md.numen.themes.presets": "Twelve ready-made color themes for the editor and the flashcards window.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Change the look of the window**: Pick Amber, Ayu, Catppuccin, Cobalt2, Dracula, GitHub, Gruvbox, Monokai, Nord, One Dark, Solarized or Tokyo Night from the appearance settings or the command palette.\n- **Keep the default**: The default Numen theme is part of the application and does not depend on this plugin.\n\n## Turning It Off\n\nDisable the plugin in the plugin settings to remove its themes from the list. A window that applies one of them falls back to the default theme.\n\n## For Authors\n\nThe plugin has no code. Its `manifest.json` lists each theme under `contributes.themes`, and each theme is one CSS file that redeclares the interface tokens. See the themes guide for the token list and the manifest fields.\n",
  "md.numen.tools.index.stats": "Shows in the title bar how many vectors the search index holds, so you can watch it grow when you add a book or a PDF or edit a note, and shrink when you delete something.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Watch the index**: The title bar shows `⬡ 12 480`, the number of chunks of text that carry a vector. While chunks still wait for their vector, the mark pulses softly in green; when none wait, it stands still. The item never shows the total of chunks: the details are in the popover.\n- **See the details**: Click the item for a list of the counts and settings behind the number. Press Escape or click elsewhere to close it.\n\nThe popover lists:\n\n- **Vectors**, **Chunks** and **Waiting for a vector**: the chunks that carry a vector, all the chunks that can, and the queue between the two.\n- **Notes and files**: how many sources the index has read, with the chunks and files of each kind: notes, PDF, EPUB and media transcripts. A kind that holds nothing is not listed.\n- **Model**, **Recipe** and **Vector size**: the embedding model in use and the dimension count of its vectors.\n- **Performance profile**: the profile you chose in the settings and the threads it grants.\n- **Embedding speed**: chunks embedded each second over the last 10 seconds, shown only while chunks wait.\n- **Last change**: when the counts last changed while the window was open.\n- **Index file size** and **Vector data size**: the disk space of the index and of the tables that hold vectors, refreshed every 30 seconds.\n\nThe plugin is enabled by default and has no settings. It only reads the index and changes nothing.\n\n## How It Works\n\nThe plugin's backend asks the host for `GetIndexStats` of `HostSearchService` (the permission `index:read`), which reads the counts with indexed queries and keeps the answer for one second. The window asks the backend every 2 seconds while it is visible and has the focus, and once when it gets the focus back. The backend derives the kind of each source from its file extension and measures the speed and the time of the last change from the answers it sees. Its service `md.numen.tools.index.stats` has the method `GetStats`.\n",
  "md.numen.tools.media.importer": "Save web articles and online videos (such as YouTube lectures) into your vault for offline reading, watching, and searching.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Import a link**: Run **Import link** with a web address to make a link note for it. An address that already has a link note in the vault opens that note instead of making a second one.\n- **Save the text of a link**: Run **Download captions** on a link note to save a clean copy of an article, or the captions of a video, as the note's text. It can be read and searched from then on.\n- **Download the media of a link**: Run **Download media** on a link note and pick **Video** or **Audio** (the sound alone) to watch or listen offline in the Media Viewer. Link notes of the same address share one download. **Delete media** takes the download off that note, and the file is deleted once no link note refers to it.\n- **Ask the agent to import a link**: The agent in the chat panel can save an address with the tool `url_import`, which makes the link note, or finds the one the address has, fetches its text and, when requested, downloads the video.\n\n## Settings\n\n- **yt-dlp**: Location of the `yt-dlp` program used to download online videos (detected automatically when installed on your system).\n- **ffmpeg**: Location of the `ffmpeg` program used to process downloaded audio and video (detected automatically when installed on your system).\n- **Caption languages**: Caption languages in order of preference, comma-separated.\n- **Automatic captions**: Use auto-generated captions when a site has no others.\n- **Download captions automatically**: Download the captions or the article of every link note added to the vault that has no text yet.\n- **Maximum download size**: The largest download in megabytes; 0 for no limit.\n",
  "md.numen.tools.proofreading": "Cleans up spelling, punctuation, and recognition errors in audio transcripts and scanned documents using an AI model—without losing timestamps or page positions.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Fix transcription and OCR mistakes**: Click **Proofread** in the Media Viewer or right-click a transcribed recording or recognized PDF in the sidebar to produce a clean, readable version.\n- **Keep the original versions**: Proofreading creates a new text version alongside the original transcript or scan recognition, so you can switch back and forth at any time.\n\n## Settings\n\n- **Connections**: Up to five connections that check the text, tried in order. Add a connection on the Connections page, then choose it here with a model. When one cannot answer, for example because its key was rejected or it ran out of credit, the next one is used. With no connection, proofreading is off.\n- **Batch size**: How many lines are proofread together at once (`40` by default).\n- **Overlap**: Extra surrounding lines included for context between batches (`4` by default).\n- **In-flight requests**: How many batches are proofread in parallel (`4` by default).\n- **Max edit distance**: Safety limit (`0.3` by default) that rejects a line if the AI rewrites too much of the original wording.\n",
  "md.numen.tools.search.relevance": "Sets how similar a note must be to your search to appear in the results. Matches below the threshold are left out of the results.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Set the threshold automatically**: Numen measures your vault once its vectors exist and uses the threshold that excludes unrelated notes.\n- **Set it yourself**: Move a slider between 0.05 and 0.95 when the automatic value does not suit your vault.\n- **Check a query**: See the best match in your vault for a search and whether it passes the threshold.\n\n## Search Relevance\n\nOpen **Search relevance** in the settings of this plugin.\n\n- **Auto / Manual**: In **Auto**, Numen picks the threshold for you. In **Manual**, a slider sets it between 0.05 and 0.95, and a marker on the slider shows the automatic value. A manual threshold applies to the current model only, so each model has its own.\n- **Threshold and Recalculate**: Shows the threshold in use. **Recalculate** measures it again on your vault.\n- **Source**: Shows where the threshold comes from: measured on your vault, the starting value of the model, a default, or set by you.\n- **Noise**: The chart shows how similar unrelated notes are to each other, with a line at the threshold. The threshold is set just above this noise, so unrelated notes are not in your results. Below the chart you see how many titles and passages were compared and when.\n- **Not enough notes yet**: Measuring needs at least 20 notes with meaning-based search indexed. Until then Numen uses the starting value of the model.\n- **Check a query**: Type a search and press **Check** to see the best match in your vault and whether it passes the threshold.\n\nChanging the model selects the threshold for the new model. If the embedding model is not running, the panel shows the error and you can try again.\n\n## When the Plugin Is Off\n\nNumen uses the last threshold the plugin set. Without one it uses the starting value of the model, and without that, 0.30. Turning the plugin on again restores your manual threshold and the last measurement.\n\n## For Plugin Authors\n\nThe plugin has the permissions `index:read`, `search:policy` and `vault:read`. It reads the note titles through the vault calls and the embedding state through `HostSearchService`, embeds titles and queries with the model in use, has the host compare the title vectors with the passages of other notes (`SampleSimilarities`), and stores the floor it chooses with `SetSearchFloor`. The host stores one floor for each vault and recipe and applies it to every search. The plugin keeps its measurements in its folder of the vault's store. The window calls the plugin through `SearchRelevanceService`.\n",
  "md.numen.tools.updater": "Shows a notice when a newer build of Numen is published, with a link to its download.\n\n<!-- more -->\n\n## What You Can Do\n\n- **See that an update exists**: A version badge appears in the title bar, after its title, while a newer build is published. Click it for the release date, the way to install on your system and a link to the release notes.\n- **Dismiss an announcement**: **Dismiss** hides the badge for that version until you restart the window. A newer version shows it again.\n\nThe plugin checks without showing messages. A machine without a network connection, or a feed that does not respond, shows nothing, and the feed is requested again later.\n\n## Settings\n\n- **Channel**: **Stable** _(default)_ notifies about releases; **Beta** notifies about builds published ahead of the next release; **Custom** asks the release feed address below.\n- **Release feed address**: Shown only for the **Custom** channel, which requires it: the address of the `latest.json` file to request the newest build. The Stable and Beta channels ask their own published feeds, and the stored address is kept while it is hidden.\n- **Check interval**: Hours between two checks (`4` by default).\n\n## How It Works\n\nThe plugin's backend requests the release feed and compares its build number with the running build; where the running build has no number, versions are compared. The window shows the result in the title bar. Its service `md.numen.tools.updater` has the methods `Check` and `GetStatus`; a check runs only after the application has configured the plugin with its settings.\n",
  "md.numen.views.agent.chat": "Open a conversation tab to ask AI assistants questions about your vault, find passages across your books and notes, and collaborate on writing.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Choose the agents on the plugin page**: Add the agents as connection rows under Agents; the chat asks them in order and uses the next one when one cannot answer.\n- **Click citations to verify sources**: Answers include clickable links to your notes, book passages, PDF pages, and transcript timestamps so you can jump straight to the source.\n- **See what the assistant is doing**: Watch each search and note lookup step as the assistant works on your request.\n- **Open the chat from the title bar**: The plugin places its Open agent chat command in the title bar as a button.\n\n## Settings\n\n- **Agents**: The connections that answer in the chat, up to five, tried in order. The model and the depth of reasoning are set on each connection row.\n",
  "md.numen.views.file.tree": "Browse, organize, rename, and manage all the notes, books, documents, and recordings in your vault from the sidebar.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Organize your library**: Create notes and folders, drag and drop files to reorganize them, and rename or delete items.\n- **Quick actions on any file**: Right-click an audio file, video, or PDF to start **Transcribe**, **Recognise text**, or **Proofread** directly from the file tree.\n- **Open the tree from the title bar**: The plugin places its Open file tree command in the title bar as a button.\n",
  "md.numen.views.flashcards.studio": "Create flashcard decks, design custom card layouts, and review cards with spaced repetition to remember what you read.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Build and manage decks**: Browse, search, and edit flashcards in a clean table view.\n- **Design custom card templates**: Define your own fields and front/back card layouts for languages, definitions, or quotes.\n- **Spaced-repetition review**: Study due cards in the dedicated review window, scheduled automatically so you review each card right before you would forget it.\n\n## Settings\n\n- **Day starts at**: The time of day (`04:00` by default) when a new daily review session begins, so late-night reviews still count toward the current day.\n- **Agents**: The connections that answer questions about a card in the review, up to five, tried in order.\n",
  "md.numen.views.media.viewer": "Play audio and video recordings and the media of link notes alongside their text, or record new voice notes directly from your microphone.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Listen and follow along**: Play recordings while the active line in the transcript or captions highlights and scrolls with the playback.\n- **Watch link notes**: A link note plays its downloaded video or audio, or shows the site's player while nothing is downloaded, above its captions or article.\n- **Jump to any moment**: Click any timestamp in the transcript to jump playback to that exact second.\n- **Transcribe, proofread, and edit by hand**: Start **Transcribe** or **Proofread** with one click in the toolbar, or click **Edit** to fix any line of the transcript yourself.\n- **Record voice notes**: Record audio from your microphone, preview it, and save it straight into your vault.\n",
  "md.numen.views.plex.graph": "Explore your notes as an interactive visual map showing how ideas connect as parent topics, subtopics, related jumps, and siblings.\n\n<!-- more -->\n\n## What You Can Do\n\n- **See the context around any note**: Shows your current note at the center with its parent topics above, subtopics below, and related notes to the side.\n- **Connect and create notes visually**: Drag a connection line from one note to another to link them, or drag onto empty space to create and link a brand-new note in that direction.\n- **Navigate the map**: Click any connected note to move the focus there and follow the links from note to note.\n- **Open the plex from the title bar**: The plugin places its Open plex graph command in the title bar as a button.\n- **Change how the plex looks**: In the plugin settings, turn off the pill behind titles, change the padding of pills, the size of the note in the middle, how long moving to another note takes, the thickness and curvature of lines, and the room around each note.\n",
  "md.numen.views.statusBar": "Draws a bar at the bottom of the window with the commands and views that plugins place at its start and at its end.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Run a command from the bar**: A command placed in the bar is a button with its icon, or its title where it has no icon. Hover over it for its title and shortcut.\n- **Open a submenu**: A submenu is a button with its title that opens the list of its entries above the bar. A click on a command runs it and closes the list; Escape or a click outside closes it too.\n- **Read what a plugin shows**: A view a plugin places in the bar is drawn in place.\n\nThe plugin is disabled by default. Disabled, the window has no bar at the bottom.\n\n## How It Works\n\nThe manifest places the view `bar` in the slot `statusBar` of `contributes.slots`, and the frontend registers it with `register.slotView`. The bar reads the entries of the slots `statusBar.left` and `statusBar.right` through `context.ui.slots` and runs their commands with `context.ui.slots.runCommand`. Entries of a group stand together, and a gap divides two groups.\n",
  "md.numen.views.tasks": "Shows the background tasks of the window in the title bar: whether any run, and the list of them with their progress.\n\n<!-- more -->\n\n## What You Can Do\n\n- **See what the window is doing**: The title bar says **Idle** while no task runs, or how many tasks are done of how many, with a ring that fills as they progress. A task that failed tints the chip red and changes no text.\n- **Open the list**: Click the summary for the list of tasks, newest first. Each row names the task, what it works on and how far it has got. Escape, a click outside or the close button closes the list.\n- **Cancel a task**: Point at a running task and click the cross. The row shows the task as cancelling until the application has stopped it.\n- **Clear a failure or a notice**: A failed task and a notice have a button that takes them out of the list. A failed task that can be started again has a retry button instead.\n- **Configure a plugin**: A failure that a setting of a plugin can repair has a link that opens the settings of that plugin.\n\nThe plugin is enabled by default. Disabled, no task status is shown anywhere in the window.\n\n## How It Works\n\nThe manifest places the view `status` in the slot `titleBar.status` of `contributes.slots`, and the frontend registers it with `register.slotView`. The view reads the tasks through `context.work.backgroundTasks`: the tasks of the core and of the plugins, the messages of the window and the notices of the plugins at warning or error level as failed tasks. Cancel, retry, dismiss and the settings link are calls of the same host.\n",
  "md.numen.views.titleBar": "Draws the bar at the top of each window: the buttons of the commands the plugins place in the title bar, the name of the vault with the list to switch vaults, and the window buttons.\n\n<!-- more -->\n\n## What You Can Do\n\n- **Run a command from the bar**: Each command a plugin places in the title bar is a button, and a submenu is a button that opens its commands. Hover over one for its name and shortcut. When the bar is too narrow, the buttons go into the menu at the end of the commands, and then the indicators at the right are hidden from the end.\n- **See what the plugins report**: The indicators the plugins place in the title bar, such as the tasks, stand at the right of the title.\n- **Use the keyboard**: Tab reaches the buttons of the bar as one stop. The arrow keys move between them, Home and End reach the first and the last.\n- **Switch vaults**: Click the name of the vault to choose another one. Below a line, the last row creates a vault, in a window that can create one.\n- **Move the window**: Drag the free space of the bar. A double click on it maximizes the window or restores it.\n- **Minimize, maximize and close the window**: On Windows the buttons are at the right, and the maximize button opens Snap Layouts. On macOS the system buttons are at the left.\n\n## Settings\n\n- **Window buttons** (`buttons`): where the bar draws the minimize, maximize and close buttons on Linux. **Right** draws square buttons as on Windows, **Left** draws three circles as on macOS. Windows and macOS keep their own place. The bar follows a change at once.\n- **Hide in full screen** (`hideInFullscreen`): off by default. When it is on and the window is in full screen, the bar is not drawn and the tabs start at the top of the window; the bar is drawn again when the window leaves full screen. The keyboard shortcut of the window still leaves full screen.\n\nThe plugin is enabled by default.\n\n## Without the Plugin\n\nDisable the plugin and the bar is gone: the tabs start at the top of the window. The window is moved by the free space of a strip of tabs at the top edge. On macOS the system buttons stay, and the strip in the top-left corner keeps room for them. On Windows and Linux the window has no buttons: close it with Alt+F4 or move it with the system menu (Alt+Space). The change needs no restart.\n\n## How It Works\n\nThe manifest places the view `bar` in the slot `titleBar` with `contributes.slots`, and the frontend registers it with `register.slotView`. The bar reads everything from its plugin context: the entries of the slots `titleBar.commands` and `titleBar.status` from `context.ui.slots`, the commands it runs from `context.ui.commands`, the vaults from `context.vault.vaults`, the state and the buttons of the window from `context.ui.window`, and the name of the application and the platform from `context.plugin.app`. It draws a command as a button with the icon of the command, a submenu as a button that opens a menu, a view of another plugin in its place, and a separator as a gap. Every entry is out of the area that drags the window. The bar draws its own window buttons, drag area and room for the macOS buttons.\n"
}
