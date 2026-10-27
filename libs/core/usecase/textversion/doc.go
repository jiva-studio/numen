// Package textversion keeps the text versions of vault files and reads their
// text from the plugin folders that hold it.
//
// A version records who made a text, from what and from which source; its
// segments are a JSON Lines file in the folder of the plugin that made it. What
// a file shows and what the index cuts is the live version, and what a file's
// menu offers follows from the enabled plugins and the versions the file has.
package textversion
