Feature: Read a PDF
  A person reads a PDF file in a tab and moves between its pages.

  As a person who reads PDF files
  I want to open a PDF file and keep my page
  So that I can come back to the page where I stopped

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: A PDF shows its page

    Scenario: A PDF shows its first page
      Given the vault has the following:
        | path             | sample    |
        | assets/paper.pdf | paper.pdf |
      When the person opens the file at "assets/paper.pdf"
      Then the PDF tab at "assets/paper.pdf" is at page "1"

    Scenario: A PDF keeps its page when the person returns
      Given the vault has the following:
        | path                   | text | sample    |
        | assets/shore.pdf       |      | shore.pdf |
        | notes/Shopping list.md | tea  |           |
      And the PDF tab at "assets/shore.pdf" is open at page "2"
      When the person opens the note titled "Shopping list"
      And the person opens the PDF tab at "assets/shore.pdf"
      Then the PDF tab at "assets/shore.pdf" is at page "2"

    Scenario: The tab moves to the other pane
      Given the vault has the following:
        | path             | sample    |
        | assets/pages.pdf | pages.pdf |
      And the PDF tab at "assets/pages.pdf" is open at page "12"
      When the person moves the PDF tab at "assets/pages.pdf" to the other pane
      Then the PDF tab at "assets/pages.pdf" is at page "12"
