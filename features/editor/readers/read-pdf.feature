Feature: Read a PDF
  A person reads a PDF file in a tab and moves between its pages.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path                   | text | sample    |
      | assets/paper.pdf       |      | paper.pdf |
      | assets/shore.pdf       |      | shore.pdf |
      | assets/pages.pdf       |      | pages.pdf |
      | notes/Shopping list.md | tea  |           |
    And the application "editor" starts

  Scenario: A PDF shows its first page
    When the person opens the file at "assets/paper.pdf"
    Then the PDF tab at "assets/paper.pdf" is at page "1"
    And the PDF tab at "assets/paper.pdf" draws page "1"

  Rule: A PDF keeps its page

    Scenario: The tab is selected again
      Given the PDF tab at "assets/shore.pdf" is open at page "2"
      When the person opens the note titled "Shopping list"
      And the person selects the PDF tab at "assets/shore.pdf"
      Then the PDF tab at "assets/shore.pdf" is at page "2"
      And the PDF tab at "assets/shore.pdf" draws page "2"

    Scenario: The tab moves to the other pane
      Given the PDF tab at "assets/pages.pdf" is open at page "12"
      When the person moves the PDF tab at "assets/pages.pdf" to the other pane
      Then the PDF tab at "assets/pages.pdf" is at page "12"
      And the PDF tab at "assets/pages.pdf" draws page "12"

  Rule: A PDF moves between pages

    Background:
      Given the person opens the file at "assets/pages.pdf"

    Scenario: A page number is typed
      When the person goes to page "12" in the PDF tab at "assets/pages.pdf"
      Then the PDF tab at "assets/pages.pdf" is at page "12"
      And the PDF tab at "assets/pages.pdf" draws page "12"
      When the application "editor" starts
      Then the PDF tab at "assets/pages.pdf" is at page "12"

    Scenario: The next page button turns the page
      When the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
      Then the PDF tab at "assets/pages.pdf" is at page "2"
      When the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
      Then the PDF tab at "assets/pages.pdf" is at page "3"
      When the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
      Then the PDF tab at "assets/pages.pdf" is at page "4"
      And the PDF tab at "assets/pages.pdf" draws page "4"

    Scenario: The buttons stop at the first and the last page
      Then the button "Next page" in the PDF tab at "assets/pages.pdf" is enabled
      And the button "Previous page" in the PDF tab at "assets/pages.pdf" is disabled
      When the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
      Then the button "Previous page" in the PDF tab at "assets/pages.pdf" is enabled
      When the person goes to page "40" in the PDF tab at "assets/pages.pdf"
      Then the button "Previous page" in the PDF tab at "assets/pages.pdf" is enabled
      And the button "Next page" in the PDF tab at "assets/pages.pdf" is disabled
