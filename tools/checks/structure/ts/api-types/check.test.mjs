import assert from "node:assert/strict";
import { test } from "node:test";
import { compareApi } from "./check.mjs";

const GO = `
type ErrorCode string

const (
	ErrorCodeUnspecified ErrorCode = ""
	ErrorCodeNotFound    ErrorCode = "not_found"
)

// Deck is a deck.
type Deck struct {
	Path   string \`json:"path"\`
	Cards  []*Card \`json:"cards"\`
	Preset *string \`json:"preset,omitempty"\`
}
`;

const TS = `
export const ErrorCode = {
  NOT_FOUND: 'not_found',
} as const

export interface Deck {
  path: string
  cards: Card[]
  preset?: string
}
`;

test("types that match produce no difference", () => {
  assert.deepEqual(compareApi({ go: GO, ts: TS, label: "p" }), []);
});

test("an enum value spelled differently is a difference", () => {
  const found = compareApi({ go: GO, ts: TS.replace("'not_found'", "'notFound'"), label: "p" });
  assert.equal(found.length, 2);
  assert.match(found.join("\n"), /ErrorCode: value not_found is missing in TypeScript/);
});

test("a field the Go struct lacks, and one it has that TypeScript lacks, are differences", () => {
  const found = compareApi({ go: GO, ts: TS.replace("  cards: Card[]\n", "  extra: string\n"), label: "p" });
  assert.match(found.join("\n"), /field cards is missing in TypeScript/);
  assert.match(found.join("\n"), /field extra is not in Go/);
});

test("a field optional on one side only is a difference", () => {
  const found = compareApi({ go: GO, ts: TS.replace("preset?: string", "preset: string"), label: "p" });
  assert.match(found.join("\n"), /Deck\.preset/);
});
