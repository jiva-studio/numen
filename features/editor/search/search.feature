Feature: Search
  A person finds notes and the passages of documents in a vault.

  Background:
    Given the application "editor" starts

  Rule: A search finds the passages of notes

    Scenario: A passage of a note is opened in its note
      Given the empty vault "Test"
      And the vault has the note titled "Entropy" with "Entropy is a measure of uncertainty."
      And the application "editor" starts
      When the person searches the vault for "Entropy"
      And the person chooses the search result "Entropy is a measure of uncertainty" in the group "Text"
      Then the tab titled "Entropy" is active
      And the note titled "Entropy" has "Entropy is a measure of uncertainty."

    Scenario: A note is found by a word it does not have
      When the person searches the vault for "boat"
      Then the search has the result "The ferry leaves the harbour" in the group "Meaning"
      And the search does not have the result "The ferry leaves the harbour" in the group "Text"

  Rule: A search finds the passages of documents

    Scenario: A passage of a PDF is opened at its page
      When the person searches the vault for "heron wades"
      And the person chooses the search result "assets/shore.pdf" in the group "Text"
      Then the tab at "assets/shore.pdf" is at page "2"

    Scenario: A passage of a PDF has the highlight
      When the person searches the vault for "shipping manifest"
      And the person chooses the search result "assets/flywheel.pdf" in the group "Text"
      Then the tab at "assets/flywheel.pdf" shows a highlight

    Scenario: A passage of a book has the highlight
      When the person searches the vault for "reflowable epub reader"
      And the person chooses the search result "assets/sample.epub" in the group "Text"
      Then the tab at "assets/sample.epub" has the highlight "reflowable"
