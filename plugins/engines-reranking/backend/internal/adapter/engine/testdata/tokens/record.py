# /// script
# requires-python = ">=3.10"
# dependencies = ["tokenizers==0.23.3"]
# ///
"""Records the reference ids of TestTokenizerMatchesReferenceIDs of engines-reranking.

Run once per change of the pairs or the pinned models, on a machine that holds the tokenizer.json files:

    uv run record.py <models folder>

The models folder holds one folder per model, named by its repository id with the slash written as two
dashes (onnx-community--bge-reranker-v2-m3-ONNX/tokenizer.json), as NUMEN_TEST_MODELS does for the test.
The script writes <folder>.json next to itself: for each pair, the query, the document, and the ids and
type ids Python `tokenizers` gives for them truncated longest_first to the 512 tokens of the profiles.
It is the recording of libs/shared/tokenizer/testdata/parity/record.py with the pairs of a reranker.
"""

import json
import os
import sys

from tokenizers import Tokenizer

MODELS = [
    ("onnx-community/gte-multilingual-reranker-base", "ee64367e35a2db0da46bb6497e13a18f8bd585cb"),
    ("onnx-community/bge-reranker-v2-m3-ONNX", "6f5ff65298512715a1e669753bc754d2bc8f367b"),
]

MAX_TOKENS = 512

LONG_DOCUMENT = " ".join(
    [
        "The lighthouse keeper writes the weather in a notebook every evening at six.",
        "On the third page she keeps a list of the ships that passed the harbor wall.",
        "Маяк стоит на скале, и смотритель поднимается по лестнице из ста двенадцати ступеней.",
        "Kṛṣṇa plays the flute in Vṛndāvana while Caitanya sings in the courtyard.",
    ]
    * 12
)

LONG_QUERY = " ".join(["Which notebook lists the ships that passed the harbor wall during the storm of the third winter?"] * 30)

PAIRS = [
    # Runs of spaces, leading and trailing spaces, a tab and a newline
    ("two  spaces", "three   spaces and ten          spaces"),
    ("  leading spaces", "trailing spaces   "),
    ("a tab\there", "a line\nand the next"),
    # Punctuation before letters
    ("(word", "«слово» —да ...and 'quoted'"),
    ("#tag @name", "[[Link]] to the garden note"),
    # IAST and Devanagari
    ("Kṛṣṇa in Vṛndāvana", "Caitanya Mahāprabhu sings in the courtyard"),
    ("कृष्ण", "कृष्ण वृन्दावन में"),
    # Full-width letters, ligatures, no-break space and the numero sign (the Precompiled normalizer)
    ("Ｆｕｌｌ－ｗｉｄｔｈ", "ﬁne ﬂow, no break, № 5"),
    # Digits and dates
    ("ferry on 2026-03-14", "The ferry leaves at 07:45 on 14.03.2026, platform 3."),
    # An empty document
    ("when does the ferry leave", ""),
    # One long document truncated longest_first, and both texts cut
    ("which notebook lists the ships", LONG_DOCUMENT),
    (LONG_QUERY, LONG_DOCUMENT),
]


def record(root, repo):
    folder = repo.replace("/", "--")
    tok = Tokenizer.from_file(os.path.join(root, folder, "tokenizer.json"))
    tok.no_padding()
    tok.enable_truncation(max_length=MAX_TOKENS, strategy="longest_first")
    out = []
    for query, document in PAIRS:
        encoding = tok.encode(query, document)
        out.append({"query": query, "document": document, "ids": encoding.ids, "typeIds": encoding.type_ids})
    target = os.path.join(os.path.dirname(os.path.abspath(__file__)), folder + ".json")
    with open(target, "w", encoding="utf-8") as file:
        json.dump(out, file, ensure_ascii=False, separators=(",", ":"))
        file.write("\n")
    print(target, len(out), "pairs")


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: uv run record.py <models folder>")
    for repo, _ in MODELS:
        record(sys.argv[1], repo)


if __name__ == "__main__":
    main()
