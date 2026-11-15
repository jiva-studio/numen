# /// script
# requires-python = ">=3.10"
# dependencies = ["tokenizers==0.23.3"]
# ///
"""Records the reference ids of TestE5TokensMatchHuggingFace.

Run once per change of the texts or of the pinned model, on a machine that holds the tokenizer.json:

    uv run record_e5_tokens.py <models folder>

The models folder holds Xenova--multilingual-e5-small/tokenizer.json, as NUMEN_TEST_MODELS does for the
test. Each text is read with the prefix of its role, as the E5 profile of the plugin sets it, and truncated
to 512 tokens as Hugging Face truncates a single text. The script writes e5-tokens.json next to itself.
"""

import hashlib
import json
import os
import sys

import tokenizers
from tokenizers import Tokenizer

REPO = "Xenova/multilingual-e5-small"
REVISION = "761b726dd34fb83930e26aab4e9ac3899aa1fa78"
MAX_LENGTH = 512
PREFIXES = {"query": "query: ", "document": "passage: "}

PARAGRAPH = (
    "A deck of cards grows as the notes grow. Each card keeps the date it is due, how stable the memory of "
    "it is and how hard it was the last time. Review takes the cards that are due, oldest first, and shows "
    "the front; the answer decides when the card comes back. Карточка, которую вспомнили легко, вернётся "
    "позже, а забытая вернётся завтра. "
)

CASES = [
    ("query", "how to schedule flashcards"),
    ("query", "как повторять карточки по расписанию"),
    ("query", "two  spaces   then    four"),
    ("query", "Ｆｕｌｌ－ｗｉｄｔｈ ｑｕｅｒｙ ０１２"),
    ("query", "non breaking space between words"),
    ("query", "Kṛṣṇa in Vṛndāvana"),
    ("query", "Заметки хранятся на диске"),
    ("document", "Spaced repetition shows a card again just before you would forget it."),
    ("document", PARAGRAPH * 12),
]


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: uv run record_e5_tokens.py <models folder>")
    path = os.path.join(sys.argv[1], REPO.replace("/", "--"), "tokenizer.json")
    with open(path, "rb") as file:
        digest = hashlib.sha256(file.read()).hexdigest()
    tok = Tokenizer.from_file(path)
    tok.enable_truncation(max_length=MAX_LENGTH)
    tok.no_padding()
    cases = []
    for role, text in CASES:
        full = tok.encode(PREFIXES[role] + text, add_special_tokens=True)
        cases.append({"role": role, "text": text, "ids": full.ids})
    tok.no_truncation()
    long_ids = tok.encode(PREFIXES["document"] + CASES[-1][1]).ids
    if len(long_ids) <= MAX_LENGTH:
        sys.exit(f"the long passage holds {len(long_ids)} tokens, not more than {MAX_LENGTH}")
    out = {
        "tokenizers": tokenizers.__version__,
        "repo": REPO,
        "revision": REVISION,
        "sha256": digest,
        "cases": cases,
    }
    target = os.path.join(os.path.dirname(os.path.abspath(__file__)), "e5-tokens.json")
    with open(target, "w", encoding="utf-8") as file:
        json.dump(out, file, ensure_ascii=False, separators=(",", ":"))
        file.write("\n")
    print(target, len(cases), "cases, the long passage holds", len(long_ids), "tokens")


if __name__ == "__main__":
    main()
