Feature: Link a new device by scanning its QR code

  A device that has the account reads the QR code of a new device. The 12 recovery words are not typed.

  Background:
    Given the empty vault "Test"
    And a sync server runs
    And the person creates an identity
    And the application "editor" starts

  Rule: A device that has the account scans the QR code of a new device

    Scenario: This device scans the QR code of a new device
      When the second computer shows the QR code to join an account
      And the person scans the QR code
      And the person approves the new device
      And the second computer enters the confirmation code of the person
      Then the second computer has the account of the person
      And the account has 2 devices

    Scenario: The device that has the account shows the new device
      When the second computer shows the QR code to join an account
      And the person scans the QR code
      Then the person sees the name and the fingerprint of the second computer

    Scenario: A wrong confirmation code refuses the new device
      When the second computer shows the QR code to join an account
      And the person scans the QR code
      And the person approves the new device
      And the second computer enters the confirmation code "000000"
      Then the second computer has no identity
      And the account has 1 device

    Scenario: Three wrong confirmation codes close the link
      Given the second computer shows the QR code to join an account
      And the person scans the QR code
      And the person approves the new device
      When the second computer enters a wrong confirmation code 3 times
      And the second computer enters the confirmation code of the person
      Then the second computer has no identity
