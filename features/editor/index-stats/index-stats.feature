Feature: Index stats
  A person sees how many vectors the index of the vault holds.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path    | text                                                       |
      | Quay.md | The ferry leaves the Quay at six. It carries tea and milk. |
    And the application "editor" starts
    And the person opens the note titled "Quay"

  Rule: The title bar shows the vectors of the notes

    Scenario: A note gives vectors
      Then the index stats show 1 vectors

    Scenario: A new note adds a vector
      When the person runs the command "Create note"
      And the person names it "Jetty"
      And the person writes "The Jetty is long. The ferry waits at the Jetty. Eggs and bread are sold at the Jetty. The boat leaves at nine." in the note titled "Jetty"
      Then the index stats show 2 vectors

    Scenario: A longer note adds a vector
      When another program writes "The ferry leaves the Quay at six. It carries tea and milk. Eggs are sold at the Quay. Bread is sold at the Quay. Salt is sold at the Quay. Fish is sold at the Quay. Rice is sold at the Quay. Oil is sold at the Quay. Wool is sold at the Quay. Rope is sold at the Quay. Tea is sold at the Quay. Soap is sold at the Quay." in the note titled "Quay"
      Then the index stats show 2 vectors

    Scenario: A removed note removes its vector
      When another program removes the note titled "Quay" from disk
      Then the index stats show 0 vectors
