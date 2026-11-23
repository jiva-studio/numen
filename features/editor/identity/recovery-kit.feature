Feature: Print and check a recovery kit
  A person keeps a printed kit of three sheets. Any two sheets restore the identity.

  As a person who uses Numen on more than one device
  I want a printed kit that I can keep in different places
  So that I get my account back when I lose all my devices and forget the words

  Background:
    Given the empty vault "Test"
    And the application "editor" starts
    And the person creates an identity

  Rule: A kit has three sheets

    Scenario: A new kit has 3 sheets
      When the person prints a recovery kit
      Then the kit has 3 sheets

    Scenario: The kit goes to the printer
      When the person prints a recovery kit
      Then the kit is sent to the printer

    Scenario: Other words are refused
      When the person prints a recovery kit with other recovery words
      Then the identity warns about "words of another account"

  Rule: Two sheets show that the kit works

    Scenario: Two sheets fit the account
      Given the person prints a recovery kit
      When the person checks the sheets "1 and 3"
      Then the identity says "the sheets fit this account"

    Scenario: One sheet is not enough
      Given the person prints a recovery kit
      When the person checks the sheets "2"
      Then the identity warns about "one more sheet"

    Scenario: A sheet with a mistake is refused
      Given the person prints a recovery kit
      When the person checks the sheets "1 and 2" with a wrong word in the sheet 2
      Then the identity warns about "sheets that are not valid"
