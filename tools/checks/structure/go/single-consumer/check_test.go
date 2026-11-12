// Package singleconsumer is the check `go/single-consumer`, run on the tree with the planted violations that prove it.
package singleconsumer

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// An RPC of the protocol or an export of an SDK that one plugin, or the plugins of one family, use and
// nothing else does belongs to that plugin or family; one that nothing outside tests uses is dead. A use
// in the Go SDK, the TypeScript SDK or the core client passes on to the consumers of the declaration
// that holds it.
func TestEveryProtocolAndSDKItemHasSeveralConsumers(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace and every TypeScript package")
	}
	checkSingleConsumers(t, load.Workspace(t))
}

var plantedFiles = map[string]string{
	"libs/core/protocol/gen/numen/v1/numenv1connect/note.connect.go": `package numenv1connect

import "context"

type NoteServiceClient interface {
	ReadNote(context.Context) error
	WriteNote(context.Context) error
	GetIndexStats(context.Context) error
	ArrangeNotes(context.Context) error
	CountWords(context.Context) error
	ListHeadings(context.Context) error
	ForgetNote(context.Context) error
	MoveNotes(context.Context) error
	TrashNotes(context.Context) error
	PinNote(context.Context) error
	TagNote(context.Context) error
	ShelveNote(context.Context) error
}

func NewNoteServiceClient() NoteServiceClient { return nil }

type noteServiceClient struct{}

func (noteServiceClient) ReadNote(context.Context) error { return nil }
`,
	"libs/plugin/sdk-go/host/host.go": `package host

import (
	"context"

	"fixture/libs/core/protocol/gen/numen/v1/numenv1connect"
)

type Client struct{ rpc numenv1connect.NoteServiceClient }

func New() *Client { return &Client{rpc: numenv1connect.NewNoteServiceClient()} }

func (c *Client) Read(ctx context.Context) error { return c.rpc.ReadNote(ctx) }

func (c *Client) Stats(ctx context.Context) error { return c.rpc.GetIndexStats(ctx) }

func (c *Client) Arrange(ctx context.Context) error { return arrange(ctx, c.rpc) }

func arrange(ctx context.Context, rpc numenv1connect.NoteServiceClient) error { return rpc.ArrangeNotes(ctx) }

func (c *Client) Move(ctx context.Context) error { return c.rpc.MoveNotes(ctx) }

func (c *Client) Trash(ctx context.Context) error { return c.rpc.TrashNotes(ctx) }

type Reader interface{ Read(context.Context) error }

func Lonely() {}

func Orphan() {}
`,
	"libs/plugin/sdk-go/plugintest/plugintest.go": `package plugintest

func Fake() {}
`,
	"libs/core/kernel/store/store.go": `package store

import (
	"context"

	"fixture/libs/core/protocol/gen/numen/v1/numenv1connect"
)

func Save(ctx context.Context) error { return numenv1connect.NewNoteServiceClient().WriteNote(ctx) }
`,
	"libs/core/kernel/store/store_test.go": `package store

import (
	"context"
	"testing"

	"fixture/libs/core/protocol/gen/numen/v1/numenv1connect"
)

func TestHeadings(t *testing.T) { _ = numenv1connect.NewNoteServiceClient().ListHeadings(context.Background()) }
`,
	"plugins/tools-stats/backend/main.go": `package main

import (
	"context"

	"fixture/libs/plugin/sdk-go/host"
	"fixture/plugins/tools-stats/backend/ports"
)

func main() {
	client := host.New()
	_ = client.Read(context.Background())
	_ = client.Stats(context.Background())
	host.Lonely()
	var mover ports.Mover = client
	_ = mover.Move(context.Background())
}
`,
	"plugins/tools-stats/backend/ports/ports.go": `package ports

import "context"

type Mover interface{ Move(context.Context) error }
`,
	"plugins/views-board/backend/main.go": `package main

import (
	"context"

	"fixture/libs/plugin/sdk-go/host"
)

type cleaner interface {
	Move(context.Context) error
	Trash(context.Context) error
}

func main() {
	var reader host.Reader = host.New()
	_ = reader.Read(context.Background())
	_ = host.New().Arrange(context.Background())
	var notes cleaner = host.New()
	_ = notes.Move(context.Background())
	_ = notes.Trash(context.Background())
}
`,
	"plugins/views-shelf/backend/main.go": `package main

import (
	"context"

	"fixture/libs/plugin/sdk-go/host"
)

func main() { _ = host.New().Arrange(context.Background()) }
`,
	"plugins/views-shelf/backend/main_test.go": `package main

import (
	"testing"

	"fixture/libs/plugin/sdk-go/host"
	"fixture/libs/plugin/sdk-go/plugintest"
)

func TestOrphan(t *testing.T) {
	host.Orphan()
	plugintest.Fake()
}
`,
	"node_modules/@connectrpc/connect/package.json": `{ "name": "@connectrpc/connect", "types": "index.d.ts" }`,
	"node_modules/@connectrpc/connect/index.d.ts": `export type Client<S extends { method: object }> = { [P in keyof S["method"]]: (request: object) => Promise<object> }
export declare function createClient<S extends { method: object }>(service: S): Client<S>
`,
	"tsconfig.base.json": `{
  "compilerOptions": {
    "strict": true,
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "noEmit": true,
    "paths": {
      "@numen/protocol": ["./libs/core/protocol/index.ts"],
      "@numen/core-client": ["./libs/core/client/src/index.ts"],
      "@numen/plugin-sdk": ["./libs/plugin/sdk-ts/src/index.ts"]
    }
  }
}`,
	"libs/core/protocol/package.json":   `{ "name": "@numen/protocol", "main": "./index.ts" }`,
	"libs/core/protocol/tsconfig.json":  `{ "compilerOptions": { "strict": true }, "include": ["**/*.ts"] }`,
	"libs/core/protocol/index.ts":       "export * from './src/note_pb'\n",
	"libs/core/protocol/src/codegen.ts": "export type GenService<T> = { method: { [K in keyof T]: T[K] } }\n",
	"libs/core/protocol/src/note_pb.ts": `import type { GenService } from './codegen'

export declare const NoteService: GenService<{
  readNote: { methodKind: 'unary' }
  countWords: { methodKind: 'unary' }
  forgetNote: { methodKind: 'unary' }
  pinNote: { methodKind: 'unary' }
  tagNote: { methodKind: 'unary' }
  shelveNote: { methodKind: 'unary' }
}>
`,
	"libs/core/client/package.json":  `{ "name": "@numen/core-client", "main": "./src/index.ts" }`,
	"libs/core/client/tsconfig.json": `{ "extends": "../../../tsconfig.base.json", "include": ["src/**/*.ts"] }`,
	"libs/core/client/src/index.ts": `import { createClient } from '@connectrpc/connect'
import { NoteService } from '@numen/protocol'

export const notes = createClient(NoteService)

export async function countAll(): Promise<object> {
  return notes.countWords({})
}

export interface PinClient {
  pin(): Promise<object>
}

export function createPinClient(): PinClient {
  return { pin: () => notes.pinNote({}) }
}

export interface TagClient {
  tag(): Promise<object>
}

class NoteTags implements TagClient {
  tag(): Promise<object> {
    return notes.tagNote({})
  }
}

export const tags: TagClient = new NoteTags()

export interface ShelfClient {
  shelve(): Promise<object>
}

export const shelf: ShelfClient = { shelve: () => notes.shelveNote({}) }
`,
	"libs/plugin/sdk-ts/package.json":  `{ "name": "@numen/plugin-sdk", "exports": { ".": "./src/index.ts", "./testing": "./src/testing/index.ts" } }`,
	"libs/plugin/sdk-ts/tsconfig.json": `{ "extends": "../../../tsconfig.base.json", "include": ["src/**/*.ts"] }`,
	"libs/plugin/sdk-ts/src/index.ts": `export { formatWords, lonelyHelper, unusedHelper } from './words'
export * from './testing/index'

export interface PluginContext {
  readNote(path: string): Promise<string>
}
`,
	"libs/plugin/sdk-ts/src/words.ts": `export function formatWords(count: number): string {
  return String(count)
}

export function lonelyHelper(): string {
  return formatWords(1)
}

export function unusedHelper(): string {
  return ''
}
`,
	"libs/plugin/sdk-ts/src/testing/index.ts": "export function createFakeContext(): object {\n  return {}\n}\n",
	"apps/editor/package.json":                `{ "name": "editor" }`,
	"apps/editor/tsconfig.json":               `{ "extends": "../../tsconfig.base.json", "include": ["src/**/*.ts"] }`,
	"apps/editor/src/main.ts": `import { createPinClient, notes, shelf } from '@numen/core-client'

export async function openNote(): Promise<object> {
  return notes.readNote({})
}

export async function pinNote(): Promise<object> {
  return createPinClient().pin()
}

interface Shelving {
  shelve(): Promise<object>
}

function keep(port: Shelving): Promise<object> {
  return port.shelve()
}

export const kept = keep(shelf)
`,
	"apps/editor/src/Count.vue": `<script setup lang="ts">
import { countAll } from '@numen/core-client'

void countAll()
</script>

<template><p>words</p></template>
`,
	"plugins/views-board/package.json":  `{ "name": "views-board" }`,
	"plugins/views-board/tsconfig.json": `{ "extends": "../../tsconfig.base.json", "include": ["frontend/**/*.ts"] }`,
	"plugins/views-board/frontend/index.ts": `import { tags } from '@numen/core-client'
import { formatWords, lonelyHelper, type PluginContext } from '@numen/plugin-sdk'

export function show(context: PluginContext): string {
  void context
  void tags.tag()
  return formatWords(2) + lonelyHelper()
}
`,
	"plugins/views-shelf/package.json":           `{ "name": "views-shelf" }`,
	"plugins/views-shelf/tsconfig.json":          `{ "extends": "../../tsconfig.base.json", "include": ["frontend/**/*.ts"] }`,
	"plugins/views-shelf/frontend/index.ts":      "import { formatWords } from '@numen/plugin-sdk'\n\nexport const label = formatWords(3)\n",
	"plugins/views-shelf/frontend/index.test.ts": "import { unusedHelper } from '@numen/plugin-sdk'\n\nexport const empty = unusedHelper()\n",
	"plugins/tools-stats/package.json":           `{ "name": "tools-stats" }`,
	"plugins/tools-stats/tsconfig.json":          `{ "extends": "../../tsconfig.base.json", "include": ["frontend/**/*.ts"] }`,
	"plugins/tools-stats/frontend/index.ts":      "import type { PluginContext } from '@numen/plugin-sdk'\n\nexport type Context = PluginContext\n",
}

func newPlantedLayout(t *testing.T, dir string) layout {
	t.Helper()
	return layout{
		root:        getRealDir(t, dir),
		modulePath:  "fixture/",
		connectPath: "fixture/libs/core/protocol/gen/numen/v1/numenv1connect",
		sdkGoPath:   "fixture/libs/plugin/sdk-go",
		protocolDir: "libs/core/protocol",
		sdkTSDir:    "libs/plugin/sdk-ts",
		libraryDirs: []string{"libs/plugin/sdk-go", "libs/plugin/sdk-ts", "libs/core/client"},
	}
}

// The walk follows a use through the SDK functions, the methods behind an interface of the SDK or of a
// plugin port, the objects and classes of the core client behind its interfaces, and a client handed to
// an interface a consumer declares, to the plugins and applications, in Go, TypeScript and Vue, and reads
// no test.
func TestSingleConsumerWalk_FindsThePlantedItems(t *testing.T) {
	files := map[string]string{}
	for name, text := range plantedFiles {
		files[name] = text
	}
	dir := load.WriteTreeFixture(t, files)
	l := newPlantedLayout(t, dir)
	goUsage := collectGoUsage([][]*packages.Package{load.LoadFixture(t, dir)}, l)
	tsUsage, read := collectTSUsage(t, l)
	if read.Files == 0 {
		t.Fatal("the TypeScript walk read no file")
	}
	var got []string
	for _, one := range findSingleConsumers([]usage{goUsage, tsUsage}, l) {
		got = append(got, one.name+": "+one.verdict)
	}
	want := []string{
		"NoteService.ArrangeNotes: used only by the views-* plugins (views-board, views-shelf): belongs to that family",
		"NoteService.ForgetNote: used by nothing outside tests: dead",
		"NoteService.GetIndexStats: used only by plugin tools-stats: belongs to that plugin",
		"NoteService.ListHeadings: used by nothing outside tests: dead",
		"NoteService.TagNote: used only by plugin views-board: belongs to that plugin",
		"NoteService.TrashNotes: used only by plugin views-board: belongs to that plugin",
		"libs/plugin/sdk-go/host.Lonely: used only by plugin tools-stats: belongs to that plugin",
		"libs/plugin/sdk-go/host.Orphan: used by nothing outside tests: dead",
		"libs/plugin/sdk-go/host.Reader: used only by plugin views-board: belongs to that plugin",
		"libs/plugin/sdk-ts/src/words.ts:formatWords: used only by the views-* plugins (views-board, views-shelf): belongs to that family",
		"libs/plugin/sdk-ts/src/words.ts:lonelyHelper: used only by plugin views-board: belongs to that plugin",
		"libs/plugin/sdk-ts/src/words.ts:unusedHelper: used by nothing outside tests: dead",
	}
	if !slices.Equal(got, want) {
		t.Errorf("findings\n%v\nwant\n%v", got, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
