Feature: Plugin settings
  A person reads and changes the settings of a plugin in the Settings tab.

  As a person who uses the editor
  I want to see the plugins in the Settings tab and change their settings
  So that I set up each plugin in one place and see only the settings that apply

  Background:
    Given the empty vault "Test"
    And the application "editor" starts

  Rule: The Settings tab shows the plugins

    Scenario: The settings list the plugins
      When the person runs the command "Open settings"
      Then the Settings tab titled "Settings" has "Audio/Video Transcription"
      And the Settings tab titled "Settings" has "Document OCR"

    Scenario: A model is shown with its languages
      Given the person runs the command "Open settings"
      Then the setting "Model" of the plugin "Audio/Video Transcription" is "Parakeet TDT 0.6B v3"
      And the setting "Model" of the plugin "Audio/Video Transcription" has the option "25 European languages"

    Scenario: A model is kept when the Settings tab is closed
      Given the person runs the command "Open settings"
      And the person sets the setting "Model" of the plugin "Audio/Video Transcription" to "Parakeet TDT 0.6B v2"
      And the person closes the Settings tab titled "Settings"
      When the person runs the command "Open settings"
      Then the setting "Model" of the plugin "Audio/Video Transcription" is "Parakeet TDT 0.6B v2"

  Rule: A setting shows the settings it needs

    Background:
      Given the person runs the command "Open settings"

    Scenario: The files of a custom model are asked for
      When the person sets the setting "Model" of the plugin "Document OCR" to "Custom model"
      Then the settings of the plugin "Document OCR" have the field "Model file"
      And the settings of the plugin "Document OCR" have the field "Character dictionary"

    Scenario: The files of a custom model are not asked for
      Given the person sets the setting "Model" of the plugin "Document OCR" to "Custom model"
      And the settings of the plugin "Document OCR" have the field "Model file"
      When the person sets the setting "Model" of the plugin "Document OCR" to "PP-OCRv5 Latin"
      Then the settings of the plugin "Document OCR" do not have the field "Model file"

    Scenario: A connection is asked for
      When the person sets the setting "Source" of the plugin "Vector Embeddings" to "Connection"
      Then the settings of the plugin "Vector Embeddings" have the field "Connection"
      And the settings of the plugin "Vector Embeddings" do not have the field "Model"
