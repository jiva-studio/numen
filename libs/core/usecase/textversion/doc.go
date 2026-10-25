// Package textversion keeps the text versions of vault files and reads their
// text from the plugins that made them.
//
// A version is who made a text, from what and from which input. The core keeps
// the record and the plugin keeps the text. What a file shows and what the index
// cuts is the live version, and what a file's menu offers follows from the
// enabled plugins and the versions the file has.
package textversion
