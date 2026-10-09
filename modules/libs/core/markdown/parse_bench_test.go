package markdown

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

var benchSampleNote = []byte(`---
title: Thermodynamics and Information Theory
status: draft
id: 01M02ACGM0FYMSXNDP29C90JNR
type: note
links:
  - to: "[[Entropy]]"
    role: parent
    type: requires
    note: fundamental concept
  - to: "[[Statistical Mechanics]]"
    role: related
---

# Thermodynamics and Information Theory

Thermodynamics is a branch of physics that deals with heat, work, and temperature.
See [[Entropy]] and [[Shannon Entropy]] for more information.

## First Law

Energy cannot be created or destroyed, only transformed from one form to another.
Refer to [[Conservation of Energy]] and [[First Law of Thermodynamics]].

~~~
code block with [[Not A Link]] inside
~~~

## Second Law

The total entropy of an isolated system always increases over time.
See [[Second Law]] as well as [[Carnot Engine]].

` + "```go\n" + `func Example() {
    // [[Also Not A Link]]
}
` + "```\n\n" + `### Microscopic Interpretation

In statistical mechanics, entropy is related to the number of microscopic configurations:
[[Boltzmann Formula|S = k ln W]] and [[Gibbs Entropy]].

## Third Law

As temperature approaches absolute zero, entropy approaches a minimum.
See [[Third Law of Thermodynamics]] and [[Absolute Zero]].
`)

var benchSampleLine = "See [[Entropy|entropy measure]] and [[Thermodynamics#Second Law]] for [[Details]]."

func BenchmarkParseNote(b *testing.B) {
	ref := domain.Fingerprint{Path: "notes/sample.md", Size: int64(len(benchSampleNote))}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = Parse(ref, benchSampleNote)
	}
}

func BenchmarkHeadings(b *testing.B) {
	_, body, _ := splitFrontmatter(benchSampleNote)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = headings(body)
	}
}

func BenchmarkWikilinksIn(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = WikilinksIn(benchSampleLine)
	}
}

func BenchmarkSplitFrontmatter(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _, _ = splitFrontmatter(benchSampleNote)
	}
}

func BenchmarkBodyLinks(b *testing.B) {
	_, body, _ := splitFrontmatter(benchSampleNote)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = bodyLinks(body)
	}
}
