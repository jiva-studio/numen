Feature: Close reader
  A person closes the tab of a PDF or a book.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path               | sample      |
      | assets/shore.pdf   | shore.pdf   |
      | assets/ledger.epub | ledger.epub |
    And the application "editor" starts

  Scenario: A PDF tab is closed
    Given the person opens the file at "assets/shore.pdf"
    And the PDF tab at "assets/shore.pdf" is open
    When the person closes the PDF tab at "assets/shore.pdf"
    Then the PDF tab at "assets/shore.pdf" is not open

  Scenario: A book tab is closed
    Given the person opens the file at "assets/ledger.epub"
    And the book tab at "assets/ledger.epub" is open
    When the person closes the book tab at "assets/ledger.epub"
    Then the book tab at "assets/ledger.epub" is not open

  Scenario: A closed PDF is opened again
    Given the person opens the file at "assets/shore.pdf"
    And the person closes the PDF tab at "assets/shore.pdf"
    When the person opens the file at "assets/shore.pdf"
    Then the PDF tab at "assets/shore.pdf" is open
