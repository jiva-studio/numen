// Package apitypes is the check `go/api-types`, run on the tree with the planted violations that prove it.
package apitypes

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// writeGoSource writes a struct tag between single quotes, since a Go raw string cannot hold a backtick.
func writeGoSource(text string) string {
	return strings.ReplaceAll(text, "'", "`")
}

func listPlantedFiles() map[string]string {
	return map[string]string{
		"libs/plugin/sdk-go/server/server.go": `package server

import "context"

type API struct{}

func Handle[Req, Res any](a *API, method string, fn func(ctx context.Context, req Req) (Res, error)) {}

func Stream[Req, Res any](a *API, method string, fn func(ctx context.Context, req Req, send func(Res) error) error) {}
`,
		"plugins/shelf/backend/api/api.go": writeGoSource(`package api

type State string

const (
	StateUnspecified State = ""
	StateOpen        State = "open"
	StateLent        State = "lent"
	StateLost        State = "lost"
)

type Format string

const (
	FormatPaper Format = "paper"
	FormatEbook Format = "ebook"
)

type Entry struct {
	Path  string 'json:"path"'
	Shelf string 'json:"shelf"'
}

type ListBooksRequest struct {
	Folder string 'json:"folder"'
}

type ListBooksResponse struct {
	Titles []string 'json:"titles"
		jsonschema:"the titles in shelf order"'
	Total int 'json:"total,omitempty"'
}

type ReadBookRequest struct {
	Path string 'json:"path"'
}

type Book struct {
	Entry
	Title string 'json:"title"'
	Pages int    'json:"pages"
		jsonschema:"the number of pages"'
	Mtime int64   'json:"mtime,string"'
	State State   'json:"state"'
	Note   *string 'json:"note,omitempty"'
	Genre  string  'json:"genre"'
	Format Format  'json:"format"'
	cache  string
}

type ReadBookResponse struct {
	Book *Book 'json:"book,omitempty"'
}

type CountBooksRequest struct{}

type CountBooksResponse struct {
	Count int 'json:"count"'
}
`),
		"plugins/shelf/backend/service/mount.go": `package service

import (
	"context"

	"fixture/libs/plugin/sdk-go/server"
	"fixture/plugins/shelf/backend/api"
)

const methodReadBook = "ReadBook"

func handle[Req, Res any](a *server.API, method string, fn func(context.Context, *Req) (*Res, error)) {
	server.Handle(a, method, func(ctx context.Context, req *Req) (*Res, error) { return fn(ctx, req) })
}

func listBooks(context.Context, *api.ListBooksRequest) (*api.ListBooksResponse, error) {
	return &api.ListBooksResponse{}, nil
}

func readBook(context.Context, api.ReadBookRequest) (api.ReadBookResponse, error) {
	return api.ReadBookResponse{}, nil
}

func countBooks(context.Context, api.CountBooksRequest) (api.CountBooksResponse, error) {
	return api.CountBooksResponse{}, nil
}

func Mount(a *server.API) {
	handle(a, "ListBooks", listBooks)
	server.Handle(a, methodReadBook, readBook)
	server.Handle(a, "CountBooks", countBooks)
}
`,
		"plugins/attic/backend/service/mount.go": writeGoSource(`package service

import (
	"context"

	"fixture/libs/plugin/sdk-go/server"
)

type GetAtticRequest struct{}

type GetAtticResponse struct {
	Count int 'json:"count"'
}

func getAttic(context.Context, GetAtticRequest) (GetAtticResponse, error) {
	return GetAtticResponse{}, nil
}

func Mount(a *server.API) {
	server.Handle(a, "GetAttic", getAttic)
}
`),
		"plugins/tools/backend/api/api.go": writeGoSource(`package api

type ListToolsInput struct {
	Query string 'json:"query"'
}
`),
		"node_modules/@numen/plugin-sdk/package.json": `{ "name": "@numen/plugin-sdk", "types": "index.d.ts" }`,
		"node_modules/@numen/plugin-sdk/index.d.ts": `export interface ApiError {
  code: string
  message: string
}
export type Result<T, E> = { ok: true; value: T } | { ok: false; error: E }
export interface Api {
  call<Req, Res>(method: string, request: Req): Promise<Result<Res, ApiError>>
  stream<Req, Res>(method: string, request: Req): AsyncGenerator<Res, ApiError | null>
}
export declare function createApi(service: string): Api
`,
		"plugins/shelf/package.json": `{ "name": "shelf" }`,
		"plugins/shelf/tsconfig.json": `{
  "compilerOptions": {
    "strict": true,
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "noEmit": true,
    "skipLibCheck": true
  },
  "include": ["frontend/**/*.ts"]
}`,
		"plugins/shelf/frontend/api/messages.ts": `export const State = {
  UNSPECIFIED: '',
  OPEN: 'open',
  LENT: 'lent',
} as const
export type State = (typeof State)[keyof typeof State]

export interface ListBooksRequest {
  folder?: string
}

export interface ListBooksResponse {
  titles: readonly string[]
  total?: number
}

export interface ReadBookRequest {
  path?: string
}

interface Book {
  path: string
  title: string
  mtime: number
  state: State
  note?: string
  genre: 'novel' | 'poem'
  format: string
  extra: boolean
}

export interface ReadBookResponse {
  book?: Book
}
`,
		"plugins/shelf/frontend/api/wire.ts": `import { createApi } from '@numen/plugin-sdk'
import type { ListBooksRequest, ListBooksResponse, ReadBookRequest, ReadBookResponse } from './messages'

const api = createApi('shelf')

const call = async <Req, Res>(method: string, request: Req): Promise<Res> => {
  const result = await api.call<Req, Res>(method, request)
  if (!result.ok) throw new Error(result.error.message)
  return result.value
}

const retry = <Req, Res>(method: string, request: Req): Promise<Res> => call<Req, Res>(method, request)

const send = (method: string, request: object) => api.call(method, request)

export const listBooks = (folder: string) => retry<ListBooksRequest, ListBooksResponse>('ListBooks', { folder })

export const readBook = (path: string) => call<ReadBookRequest, ReadBookResponse>('ReadBook', { path })

export const burnBook = (path: string) => call<{ path: string }, Record<string, never>>('BurnBook', { path })

export const lendBook = (path: string) => send('LendBook', { path })
`,
		"plugins/shelf/frontend/ui/ShelfCount.vue": `<script setup lang="ts">
import { createApi } from '@numen/plugin-sdk'

const api = createApi('shelf')
void api.call<Record<string, never>, { count: string }>('CountBooks', {})
</script>

<template>
  <p>shelf</p>
</template>
`,
		"plugins/shelf/frontend/api/wire.test.ts": `import { createApi } from '@numen/plugin-sdk'

export const probe = () => createApi('shelf').call<{ nothing: true }, { nothing: true }>('ListBooks', { nothing: true })
`,
	}
}

// The rule reads the JSON fields Go sends as reflect reads the tags, also from an embedded struct and from
// a tag on several lines, reads the calls in the script of a Vue file, follows a method through the
// functions that pass it on, on both sides, and fails a field, a value, an enumeration or an optionality
// on one side only, a method the backend does not serve, a call
// whose types it cannot name, and a plugin that serves methods its frontend never calls. Test files are
// out of scope, and a plugin that serves no method has nothing to compare.
func TestAPITypesRule_RejectsDifferencesBetweenGoAndTypeScript(t *testing.T) {
	dir := load.WriteTreeFixture(t, listPlantedFiles())
	l := layout{root: getRealDir(t, dir), modulePath: "fixture/", serverPath: "fixture/libs/plugin/sdk-go/server"}
	findings, count := runAPITypes(t, load.LoadFixture(t, dir), l)
	var got []string
	for _, one := range findings {
		got = append(got, one.text)
	}
	slices.Sort(got)
	want := []string{
		"CountBooks response.count: Go has a number, TypeScript has a string",
		"ReadBook response.book.format: Go has an enumeration (ebook, paper), TypeScript has a plain string",
		"ReadBook response.book.genre: TypeScript has an enumeration (novel, poem), Go has a plain string",
		"ReadBook response.book.mtime: Go has a string, TypeScript has a number",
		"ReadBook response.book.state: value lost is missing in TypeScript",
		"ReadBook response.book: field extra is not in Go",
		"ReadBook response.book: field pages is missing in TypeScript",
		"ReadBook response.book: field shelf is missing in TypeScript",
		"the backend serves 1 JSON method(s) and the frontend calls none of them through the Api of @numen/plugin-sdk",
		"the frontend calls BurnBook, which the backend does not serve",
		"the frontend calls a method whose name or types the type checker cannot read: the call of the Api gives no type arguments for its request and response",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if count.methods != 4 || count.calls != 4 {
		t.Errorf("%d served methods and %d frontend calls read, want the 4 methods and the 4 calls of the fixture", count.methods, count.calls)
	}
}

func TestPluginAPITypesMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the plugin modules and every plugin frontend")
	}
	checkPluginAPITypes(t)
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
