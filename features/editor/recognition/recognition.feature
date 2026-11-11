Feature: Recognition
  A person makes the text of a scan and finds it.

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A scan is found by its text

    Background:
      Given the vault has the following:
        | path                            | sample                   |
        | assets/regenerative-braking.pdf | regenerative-braking.pdf |
      And the person recognizes the text of the file at "assets/regenerative-braking.pdf"

    Scenario: A scan is found by its words
      When the person searches the vault for "recovers kinetic energy"
      Then the search has the result "assets/regenerative-braking.pdf" in the group "Text"

    Scenario: A passage opens with the highlight
      Given the person searches the vault for "recovers kinetic energy"
      When the person chooses the search result "assets/regenerative-braking.pdf" in the group "Text"
      Then the tab at "assets/regenerative-braking.pdf" shows a highlight

  Rule: A scan is found without Document OCR

    Background:
      Given the vault has the following:
        | path                | sample       |
        | assets/flywheel.pdf | flywheel.pdf |
      And the person recognizes the text of the file at "assets/flywheel.pdf"

    Scenario: The recognized text is found
      When the person disables the plugin "Document OCR"
      And the person searches the vault for "rotor spinning in a vacuum"
      Then the search has the result "assets/flywheel.pdf" in the group "Text"

    Scenario: The proofread text is found
      Given the person proofreads the recognized text of the file at "assets/flywheel.pdf"
      When the person disables the plugin "Document OCR"
      And the person searches the vault for "proofread"
      Then the search has the result "assets/flywheel.pdf" in the group "Text"
