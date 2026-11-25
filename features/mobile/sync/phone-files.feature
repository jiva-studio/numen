Feature: See the whole vault on the phone

  The phone lists every file of a synced vault.
  A note comes at once. A large file comes when the person opens it.

  As a person who uses Numen on a phone
  I want to see all my files without filling the phone
  So that I can open a book or a note when I need it

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts
    And the person syncs the vault "Test"
    And the phone is connected to the synced vault "Test"

  Rule: The phone lists every file of the synced vault

    Scenario: A note comes to the phone with its text
      When the person writes the note "Shopping list" with the text "milk" in the vault "Test"
      Then the phone has the note "Shopping list" with the text "milk"

    Scenario: A large file is listed and is not downloaded
      When the person writes the file "Books/Guide.pdf" of 5 MB in the vault "Test"
      Then the phone lists the file "Books/Guide.pdf"
      And the phone has not downloaded the file "Books/Guide.pdf"

    Scenario: A moved file is listed in its new place
      Given the person writes the file "Books/Guide.pdf" of 5 MB in the vault "Test"
      When the person moves the file "Books/Guide.pdf" to "Archive/Guide.pdf" in the vault "Test"
      Then the phone lists the file "Archive/Guide.pdf"
      And the phone does not list the file "Books/Guide.pdf"

    Scenario: A deleted file leaves the list
      Given the person writes the file "Books/Guide.pdf" of 5 MB in the vault "Test"
      When the person deletes the file "Books/Guide.pdf" in the vault "Test"
      Then the phone does not list the file "Books/Guide.pdf"

  Rule: A large file is downloaded when the person opens it

    Scenario: An opened file is downloaded
      Given the person writes the file "Books/Guide.pdf" of 5 MB in the vault "Test"
      When the person opens the file "Books/Guide.pdf" on the phone
      Then the phone has downloaded the file "Books/Guide.pdf"
      And the phone shows the file "Books/Guide.pdf"

    Scenario: A file is not downloaded while the server is away
      Given the person writes the file "Books/Guide.pdf" of 5 MB in the vault "Test"
      And the sync server stops
      When the person opens the file "Books/Guide.pdf" on the phone
      Then the phone warns about "Could not reach the sync server"
      And the phone has not downloaded the file "Books/Guide.pdf"

  Rule: A move on the phone is a move in the vault

    Scenario: A file that was not downloaded is moved on the phone
      Given the person writes the file "Books/Guide.pdf" of 5 MB in the vault "Test"
      When the person moves the file "Books/Guide.pdf" to "Archive/Guide.pdf" on the phone
      Then the vault "Test" has the file "Archive/Guide.pdf"
      And the vault "Test" does not have the file "Books/Guide.pdf"
