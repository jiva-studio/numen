Feature: Turn PDF pages
  A person goes to a page of a PDF by its number or with the buttons.

  Background:
    Given the empty vault "Test"
    And the vault has the following:
      | path             | sample    |
      | assets/pages.pdf | pages.pdf |
    And the application "editor" starts
    And the person opens the file at "assets/pages.pdf"

  Scenario: A page number is typed
    When the person goes to page "12" in the PDF tab at "assets/pages.pdf"
    Then the PDF tab at "assets/pages.pdf" is at page "12"

  Scenario: A typed page stays after the application starts again
    Given the person goes to page "12" in the PDF tab at "assets/pages.pdf"
    When the application "editor" restarts
    Then the PDF tab at "assets/pages.pdf" is at page "12"

  Scenario: The next page button turns the page
    Given the button "Next page" in the PDF tab at "assets/pages.pdf" is enabled
    And the button "Previous page" in the PDF tab at "assets/pages.pdf" is disabled
    When the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
    Then the PDF tab at "assets/pages.pdf" is at page "2"
    And the button "Previous page" in the PDF tab at "assets/pages.pdf" is enabled

  Scenario: The previous page button returns to the first page
    Given the person clicks the button "Next page" in the PDF tab at "assets/pages.pdf"
    And the PDF tab at "assets/pages.pdf" is at page "2"
    When the person clicks the button "Previous page" in the PDF tab at "assets/pages.pdf"
    Then the PDF tab at "assets/pages.pdf" is at page "1"

  Scenario: The buttons stop at the last of 40 pages
    When the person goes to page "40" in the PDF tab at "assets/pages.pdf"
    Then the button "Previous page" in the PDF tab at "assets/pages.pdf" is enabled
    And the button "Next page" in the PDF tab at "assets/pages.pdf" is disabled
