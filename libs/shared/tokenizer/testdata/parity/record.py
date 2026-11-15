# /// script
# requires-python = ">=3.10"
# dependencies = ["tokenizers==0.23.3"]
# ///
"""Records the reference ids of the parity test of libs/shared/tokenizer.

Run once per change of the texts or the pinned models, on a machine that holds the tokenizer.json files:

    uv run record.py <models folder>

The models folder holds one folder per model, named by its repository id with the slash written as two
dashes (Xenova--bge-m3/tokenizer.json), as NUMEN_TEST_MODELS does for the test. The script writes
<folder>.json next to itself: the sha256 of the tokenizer.json it read, the ids of each single text and
the ids and type ids of each pair.
"""

import hashlib
import json
import os
import sys

import tokenizers
from tokenizers import Tokenizer

MODELS = [
    ("Xenova/multilingual-e5-small", "761b726dd34fb83930e26aab4e9ac3899aa1fa78"),
    ("Xenova/bge-m3", "4de13258303883538bd53b696b452bf8099f0858"),
    ("onnx-community/gte-multilingual-reranker-base", "ee64367e35a2db0da46bb6497e13a18f8bd585cb"),
    ("onnx-community/bge-reranker-v2-m3-ONNX", "6f5ff65298512715a1e669753bc754d2bc8f367b"),
]

# Two samples of the markdown of this repository: README.md and docs/plugins/README.md.
README = """## Why Numen

- **Notes** — plain Markdown files on your disk, alongside the PDFs, EPUBs, recordings and videos they come from.
- **Graph** — typed links between notes, shown as a live graph around the note you are reading.
- **Agent** — ask your vault a question and get an answer that links to the notes it used.
- **Spaced repetition** — flashcards written inside your notes, scheduled with FSRS.
## Community

- [User Manual](https://docs.numen.md) — setting up a vault, linking, cards and review.
- [Forum](https://forum.numen.md) — workflows, questions and feature requests.
- [Issues](https://github.com/jiva-studio/numen/issues) — bug reports.
"""

PLUGINS = """The host creates two directories for each plugin before it starts the process, readable by the user alone and shared with no other plugin, and two that every plugin shares. It sets them in the environment: `NUMEN_PLUGIN_CACHE_DIR` for what the plugin can compute again, `NUMEN_PLUGIN_DATA_DIR` for what it cannot and the user would lose, `NUMEN_SHARED_CACHE_DIR` for what every plugin can compute again (compiled CoreML models), and `NUMEN_SHARED_DATA_DIR` for what every plugin downloads or makes and cannot compute again (models). The private ones are `<cache root>/plugins/<plugin id>` and `<data root>/plugins/<plugin id>` under the cache and data folders of the platform; the shared ones are `<cache root>/shared` and `<data root>/shared`. Models are in the data directory because the operating system may purge a cache and a model is gigabytes. The host removes a plugin's cache directory when the plugin is uninstalled or its version changes, keeps its data directory, and never deletes either shared directory. A plugin never computes a platform path: in Go it calls `cache.GetPluginDir()`, `cache.GetDataDir()`, `cache.GetSharedDir()` and `cache.GetSharedDataDir()` of `plugin/sdk-go`, which return an error where the host gave no directory, `cache.GetSharedDirOrPluginDir()` and `cache.GetSharedDataDirOrPluginDataDir()` for a host too old to share one, and `plugintest.GrantStorage(t)` grants all four in a test.

The shared directories hold one copy of what several plugins use, so a model of 8 GB is stored once and not once for each plugin. The rules are those of a content-addressed store: an entry is named by its digest (`models/` in the data directory for the models the host provisions, `coreml/<model digest>/` in the cache directory for the compiled form CoreML makes of a model); a writer writes a temporary file in the same directory, verifies the digest and renames it into place, and a reader that finds the digest uses it without hashing it again; two writers of one entry take `shared.Lock(dir)` of `plugin/sdk-go/shared`; no plugin deletes from either. The models folder is `<data root>/shared/models`, and `NUMEN_MODELS_DIR` set in the environment of the application moves it elsewhere; the host passes the folder it resolved to every plugin as `NUMEN_MODELS_DIR`. Nothing is read from an earlier location: a model in `~/.numen/dependencies` or `<config>/numen/models` is ignored and downloaded again.
"""

TEXTS = [
    # English
    "How do I link two notes so the graph shows the connection?",
    "query: how to schedule flashcards",
    "passage: Spaced repetition shows a card again just before you would forget it.",
    # Russian
    "Заметки хранятся на диске в виде обычных файлов Markdown.",
    "Как связать две заметки, чтобы граф показал связь?",
    # IAST and Devanagari
    "Kṛṣṇa in Vṛndāvana; Caitanya Mahāprabhu; jñāna, ṛṣi, śloka.",
    "कृष्ण वृन्दावन में",
    # Runs of spaces, tabs and newlines
    "two  spaces, three   spaces, ten          spaces",
    "  leading and trailing  ",
    "tab\there and a\nnewline\r\nand CRLF",
    "\t\n  \t",
    # Punctuation before letters
    "(word «слово» —да ...and 'quoted' #tag @name [[Link]] {curly} <angle>",
    "\"double\" „Russian“ ¿qué? ¡hola! *bold* _under_ `code` ~strike~",
    # NFKC-sensitive characters (the Precompiled charsmap)
    "Ｆｕｌｌ－ｗｉｄｔｈ Ｌａｔｉｎ ０１２３",
    "ﬁne ﬂow ﬀ ﬃ",
    "non breaking narrow thin　ideographic",
    "№ 5, ① ② ③, ㎏ ㎞, ½ ¼, ™ ©, ℃",
    "café vs café, ñ vs ñ, Å vs Å",
    "é̂̃̄ stacked marks",
    "zero​width‌joiner‍and﻿bom",
    "control\u0007bell\u001b[0m and \u0000nul",
    # CJK and emoji
    "東京で日本語のノートを書く。中文笔记。한국어 메모.",
    "Thumbs 👍🏽 family 👨‍👩‍👧 flag 🇮🇳 heart ❤️",
    # Digits, dates and code
    "2026-11-14 at 15:42, version 1.25.0, 3.14159, -42, 1e10",
    "func Encode(text string) []int { return nil }",
    # Empty
    "",
    # Repository markdown
    README,
    PLUGINS,
]

QUERY_EN = "where are the shared models stored"
QUERY_RU = "где хранятся общие модели"
DOC_SHORT = "Models are in the data directory because the operating system may purge a cache."

PAIRS = [
    # (first, second, max_length)
    (QUERY_EN, DOC_SHORT, 512),
    (QUERY_RU, "Модели лежат в каталоге данных, потому что система может очистить кэш.", 512),
    ("query: Kṛṣṇa", "passage: Vṛndāvana  is   a town.", 512),
    (QUERY_EN, PLUGINS, 512),
    (PLUGINS, QUERY_EN, 512),
    (PLUGINS, README + PLUGINS, 512),
    (README, PLUGINS, 512),
    (QUERY_EN, PLUGINS, 64),
    (PLUGINS, README, 63),
    (README, README, 63),
    (QUERY_EN, DOC_SHORT, 16),
    (QUERY_EN, DOC_SHORT, 17),
    (DOC_SHORT, QUERY_EN, 17),
    ("", DOC_SHORT, 32),
    (QUERY_EN, "", 32),
    ("", "", 8),
    (QUERY_EN, DOC_SHORT, 6),
    (QUERY_EN, DOC_SHORT, 5),
    (QUERY_EN, DOC_SHORT, 4),
    # A max_length below the 4 special tokens is left out: tokenizers then does not truncate at all,
    # where Tokenizer.EncodePair returns ErrBudgetTooSmall.
]


def record(root, repo, revision):
    folder = repo.replace("/", "--")
    path = os.path.join(root, folder, "tokenizer.json")
    with open(path, "rb") as file:
        digest = hashlib.sha256(file.read()).hexdigest()
    tok = Tokenizer.from_file(path)
    tok.no_truncation()
    tok.no_padding()
    singles = [{"text": text, "ids": tok.encode(text).ids} for text in TEXTS]
    pairs = []
    for first, second, max_length in PAIRS:
        tok.enable_truncation(max_length=max_length, strategy="longest_first")
        encoding = tok.encode(first, second)
        pairs.append({"first": first, "second": second, "maxTokens": max_length, "ids": encoding.ids, "typeIds": encoding.type_ids})
    tok.no_truncation()
    out = {
        "tokenizers": tokenizers.__version__,
        "repo": repo,
        "revision": revision,
        "sha256": digest,
        "singles": singles,
        "pairs": pairs,
    }
    target = os.path.join(os.path.dirname(os.path.abspath(__file__)), folder + ".json")
    with open(target, "w", encoding="utf-8") as file:
        json.dump(out, file, ensure_ascii=False, separators=(",", ":"))
        file.write("\n")
    print(target, len(singles), "texts", len(pairs), "pairs")


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: uv run record.py <models folder>")
    for repo, revision in MODELS:
        record(sys.argv[1], repo, revision)


if __name__ == "__main__":
    main()
