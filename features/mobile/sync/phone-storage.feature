Feature: Free space on the phone

  The person removes the download of a file and keeps the file in the vault.

  As a person who uses Numen on a phone
  I want to remove a large file from the phone
  So that I get the space back and lose nothing

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And the phone is connected to the synced vault "Test"
    And the person writes the file "Books/Guide.pdf" of 5 MB in the vault "Test"
    And the person opens the file "Books/Guide.pdf" on the phone

  Rule: A removed download is not a deleted file

    Scenario: The phone still lists a file after its download is removed
      When the person removes the download of the file "Books/Guide.pdf" on the phone
      Then the phone lists the file "Books/Guide.pdf"
      And the phone has not downloaded the file "Books/Guide.pdf"

    Scenario: The vault keeps a file after its download is removed on the phone
      When the person removes the download of the file "Books/Guide.pdf" on the phone
      Then the vault "Test" has the file "Books/Guide.pdf"
      And the sync server holds the file "Books/Guide.pdf"

    Scenario: A file downloads again after its download is removed
      Given the person removes the download of the file "Books/Guide.pdf" on the phone
      When the person opens the file "Books/Guide.pdf" on the phone
      Then the phone has downloaded the file "Books/Guide.pdf"

  Rule: A delete on the phone is a delete in the vault

    Scenario: A file deleted on the phone leaves the vault
      When the person deletes the file "Books/Guide.pdf" on the phone
      Then the vault "Test" does not have the file "Books/Guide.pdf"
