Feature: Restore an identity
  A person deletes the identity from this device and gets it back from its 12 recovery words.

  As a person who uses Numen on more than one device
  I want an identity that 12 words restore
  So that my account is the same after I lose this device

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

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
