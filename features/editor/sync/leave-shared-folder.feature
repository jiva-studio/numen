@sync-server
Feature: Leave a shared folder

  A reader leaves a folder another person shares. It stops receiving changes and keeps the notes it has.

  As a person who reads a shared folder
  I want to leave it
  So that it stops changing on my device and stays there as an ordinary folder

  Rule: The files stay

    Scenario: The reader leaves and keeps the notes
      Given the empty vault "Test"
      And a sync server runs
      And the person has an identity
      And the application "editor" starts
      And the person syncs the vault "Test"
      And the person writes the note "Food/Shopping list" with the text "milk" in the vault "Test"
      And the reader sets up an identity
      And the person shares the folder "Food" of the vault "Test" with the reader
      And the reader has the note "Shopping list" with the text "milk" in the shared vault "Food"
      When the reader leaves the shared vault "Food"
      Then the reader has no vault "Food"
      And the folder of the shared vault "Food" of the reader still has the file "Shopping list.md" with the text "milk"
