Feature: Create and restore an identity
  A person keeps an account identity on this device and gets it back from 12 recovery words.

  As a person who uses Numen on more than one device
  I want an identity that 12 words restore
  So that my account is the same after I lose this device

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The settings have the page "Identity and licenses"

    Scenario: The settings list the page "Identity and licenses"
      When the person runs the command "Open settings"
      Then the Settings tab titled "Settings" has "Identity and licenses"
      And the Settings tab titled "Settings" does not have "Licenses"

  Rule: A new identity shows its words once, then checks three of them

    Scenario: A new identity has 12 recovery words
      When the person starts a new identity
      Then the identity shows 12 recovery words

    Scenario: The words are checked before the identity is kept
      Given the person starts a new identity
      When the person writes the recovery words down
      Then the identity asks for 3 of the recovery words
      And this device has no identity

    Scenario: The identity is kept after three words are confirmed
      Given the person starts a new identity
      And the person writes the recovery words down
      When the person confirms 3 of the recovery words
      Then this device has an identity

    Scenario: A wrong word keeps no identity
      Given the person starts a new identity
      And the person writes the recovery words down
      When the person confirms the recovery words with the word "zoo" in place of one word
      Then this device has no identity

    Scenario: The words are not shown again
      Given the person creates an identity
      When the person opens the page "Identity and licenses"
      Then the identity does not show the recovery words

  Rule: The words restore the same account

    Scenario: A deleted identity leaves this device
      Given the person creates an identity
      When the person deletes the identity from this device
      Then this device has no identity

    Scenario: The words give back the same account
      Given the person creates an identity
      And the person deletes the identity from this device
      When the person restores the identity from its recovery words
      Then the account fingerprint is the same as before

    Scenario: Words with a mistake are refused
      When the person restores an identity from the word "abandon" written 12 times
      Then the identity warns about "words that are not valid"
      And this device has no identity
