Feature: Search
  A person finds notes and the passages of documents in a vault.

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
        | path              | text                                                                   |
        | notes/crossing.md | # Crossing\n\nThe ferry leaves the harbour at dawn with the morning post. |
      When the person searches the vault for "boat"
      Then the search has the result "The ferry leaves the harbour" in the group "Meaning"
      And the search does not have the result "The ferry leaves the harbour" in the group "Text"

  Rule: Search finds documents

    Scenario: A PDF opens at its page
      Given the vault has the following:
        | path             | sample    |
        | assets/shore.pdf | shore.pdf |
      When the person searches the vault for "heron wades"
      And the person chooses the search result "assets/shore.pdf" in the group "Text"
      Then the tab at "assets/shore.pdf" is at page "2"

    Scenario: A PDF passage is highlighted
      Given the vault has the following:
        | path                | sample       |
        | assets/flywheel.pdf | flywheel.pdf |
      When the person searches the vault for "shipping manifest"
      And the person chooses the search result "assets/flywheel.pdf" in the group "Text"
      Then the tab at "assets/flywheel.pdf" shows a highlight

    Scenario: A book passage is highlighted
      Given the vault has the following:
        | path               | sample      |
        | assets/sample.epub | sample.epub |
      When the person searches the vault for "reflowable epub reader"
      And the person chooses the search result "assets/sample.epub" in the group "Text"
      Then the tab at "assets/sample.epub" has the highlight "reflowable"

    Scenario: A book passage has one action
      Given the vault has the following:
        | path               | sample      |
        | assets/sample.epub | sample.epub |
      When the person searches the vault for "reflowable epub reader"
      Then the search result "assets/sample.epub" in the group "Text" has only the following actions:
        | action                 |
        | Open the document here |

    Scenario: A book passage is opened with its action
      Given the vault has the following:
        | path               | sample      |
        | assets/sample.epub | sample.epub |
      When the person searches the vault for "reflowable epub reader"
      And the person opens the actions of the search result "assets/sample.epub" in the group "Text"
      And the person chooses the action "Open the document here"
      Then the tab at "assets/sample.epub" has the highlight "reflowable"
