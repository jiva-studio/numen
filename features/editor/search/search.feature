Feature: Search
  A person finds the notes of a vault.

  As a person who keeps notes
  I want to search my notes by their words or by their meaning
  So that I can find a note without knowing where it is

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: Search finds notes

    Scenario: A passage opens its note
      Given the vault has the following:
        | path       | text                                 |
        | Entropy.md | Entropy is a measure of uncertainty. |
      When the person searches the vault for "Entropy"
      And the person chooses the search result "Entropy is a measure of uncertainty" in the group "Text"
      Then the note tab at "Entropy.md" is active
      And the note titled "Entropy" has "Entropy is a measure of uncertainty."

    Scenario: A note is found by meaning
      Given the vault has the following:
        | path          | text                  |
        | notes/Boat.md | A boat leaves at six. |
      When the person searches the vault for "ship"
      Then the search has the result "A boat leaves at six" in the group "Meaning"
      And the search does not have the result "A boat leaves at six" in the group "Text"

    Scenario: A word nobody wrote has no result
      Given the vault has the following:
        | path       | text                                 |
        | Entropy.md | Entropy is a measure of uncertainty. |
      And the person searches the vault for "measure"
      And the search has the result "Entropy is a measure of uncertainty" in the group "Text"
      When the person searches the vault for "zebra"
      Then the search does not have the result "Entropy is a measure of uncertainty" in the group "Text"

    Scenario: A result changes after the note is edited
      Given the vault has the following:
        | path             | text |
        | Shopping list.md | milk |
      And the person opens the note titled "Shopping list"
      And the person searches the vault for "milk"
      And the search has the result "milk" in the group "Text"
      When the person replaces "milk" with "bread" in the note titled "Shopping list"
      And the person searches the vault for "bread"
      Then the search has the result "bread" in the group "Text"
      And the search does not have the result "milk" in the group "Text"

  Rule: A found note has actions

    Background:
      Given the vault has the following:
        | path          | text                        |
        | notes/Quay.md | The quay is at the harbour. |

    Scenario: A found note can be shown in the plex
      When the person searches the vault for "Quay"
      Then the search result "Quay" in the group "Names" contains the following actions:
        | action       |
        | Open note    |
        | Show in plex |

    Scenario: A found note has no plex action
      Given the person disables the plugin "Plex Graph"
      When the person searches the vault for "Quay"
      Then the search result "Quay" in the group "Names" has only the following actions:
        | action             |
        | Open note          |
        | Show in file tree  |
        | Create note “Quay” |

    Scenario: A found note is opened without the plex
      Given the person disables the plugin "Plex Graph"
      When the person searches the vault for "Quay"
      And the person chooses the search result "Quay" in the group "Names"
      Then the tab at "notes/Quay.md" is active
