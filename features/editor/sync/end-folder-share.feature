Feature: End the share of a folder

  A person ends the share of a folder. The reader keeps what it has and gets nothing more.

  As a person who keeps notes in Numen
  I want to stop another person from getting more of my notes of one folder
  So that the notes I write later stay mine

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And the person writes the note "Food/Shopping list" with the text "milk" in the vault "Test"
    And the person writes the note "Plan" with the text "tea" in the vault "Test"
    And the reader sets up an identity

  Rule: The person ends the share

    Scenario: The reader keeps what it has and gets no more
      Given the person shares the folder "Food" of the vault "Test" with the reader
      And the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"
      When the person removes the reader from the sharing of the folder "Food"
      And the person writes the note "Food/Menu" with the text "soup" in the vault "Test"
      Then the reader is told "The owner ended this share" in the shared vault "Food"
      And the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"
      And the reader does not have the file "Menu.md" in the shared vault "Food"

    # Pending: needs the move of a file (#2569) and its delete path (#2570) in the sync plugin.
    @skip
    Scenario: A note moved out of the folder leaves the reader
      Given the person shares the folder "Food" of the vault "Test" with the reader
      And the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"
      When the person moves the file "Food/Shopping list.md" to "Shopping list.md" in the vault "Test"
      Then the reader does not have the file "Shopping list.md" in the shared vault "Food"
