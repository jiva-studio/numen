Feature: Share a folder with another person

  A person shares a folder of a synced vault. The reader sees the folder as a vault of its own and only reads it.

  As a person who keeps notes in Numen
  I want to give another person my notes of one folder to read
  So that we use the same notes and nothing else of mine is shown

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person connects the vault "Test" as a new synced vault
    And the person writes the note "Food/Shopping list" with the text "milk" in the vault "Test"
    And the person writes the note "Plan" with the text "tea" in the vault "Test"
    And the reader creates an identity

  Rule: The reader gets the shared folder as a vault of its own

    Scenario: The reader has the notes of the shared folder
      When the person shares the folder "Food" of the vault "Test" with the reader
      Then the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"

    Scenario: The reader does not have the notes outside the folder
      When the person shares the folder "Food" of the vault "Test" with the reader
      Then the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"
      And the reader does not have the file "Plan.md" in the shared vault "Food"
      And the reader has no vault "Test"

    Scenario: The folder stays in the vault of the person
      When the person shares the folder "Food" of the vault "Test" with the reader
      Then the vault "Test" has the note "Food/Shopping list" with the text "milk"

    Scenario: A new note of the folder arrives
      Given the person shares the folder "Food" of the vault "Test" with the reader
      When the person writes the note "Food/Menu" with the text "soup" in the vault "Test"
      Then the reader has the note "Menu" with the text "soup" in the shared vault "Food"

    Scenario: The sync server cannot read the shared notes
      When the person shares the folder "Food" of the vault "Test" with the reader
      And the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"
      Then the sync server holds no stored text "milk"

  Rule: The person sees who reads the folder

    Scenario: The share lists the reader with its fingerprint
      When the person shares the folder "Food" of the vault "Test" with the reader
      Then the sharing of the folder "Food" lists the reader with its account fingerprint

    Scenario: An invitation of another person is refused
      When the person shares the folder "Food" of the vault "Test" with the reader
      And the invitation is given to a second reader
      Then the second reader is told "This invitation is for another person"
