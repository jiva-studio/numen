Feature: Search after an edit
  A person finds the words a note has now and not the words it had.

  As a person who keeps notes
  I want the search to follow my edits
  So that a result is a text the note holds

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | text |
      | Shopping list.md | milk |
    And the application "editor" starts
    And the person opens the note titled "Shopping list"
    And the person searches the vault for "milk"
    And the search has the result "milk" in the group "Text"
    And the person replaces "milk" with "bread" in the note titled "Shopping list"

  Scenario: A word that was removed has no result
    When the person searches the vault for "milk"
    Then the search does not have the result "milk" in the group "Text"

  Scenario: A word that was added has a result
    When the person searches the vault for "bread"
    Then the search has the result "bread" in the group "Text"
