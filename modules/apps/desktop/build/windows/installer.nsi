; The Windows installer. Everything it is told is given on the command line by
; the packaging workflow.

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
; Windows reads a version as four numbers, so it is given one alongside the
; name the person sees.
!ifndef NUMERIC_VERSION
  !define NUMERIC_VERSION "0.0.0.0"
!endif
!ifndef BINARY
  !error "BINARY is the numen.exe to install"
!endif
; The two windows ship together, one version, one installer.
!ifndef FLASHCARDS
  !error "FLASHCARDS is the numen-flashcards.exe to install"
!endif
; The flashcards window wears a mark of its own. The drawing is installed
; beside the program, and the Start Menu shortcut points at it there.
!ifndef FLASHCARDS_ICON
  !error "FLASHCARDS_ICON is the numen-flashcards.ico to install"
!endif
!ifndef WEBVIEW2
  !error "WEBVIEW2 is the Microsoft bootstrapper to carry"
!endif
!ifndef OUTFILE
  !error "OUTFILE is the installer to write"
!endif

!include "MUI2.nsh"
!include "x64.nsh"
!include "LogicLib.nsh"

Name "Numen"
OutFile "${OUTFILE}"
Unicode true
RequestExecutionLevel admin
InstallDir "$PROGRAMFILES64\Numen"
InstallDirRegKey HKLM "Software\Numen" "InstallDir"

VIProductVersion "${NUMERIC_VERSION}"
VIAddVersionKey "ProductName" "Numen"
VIAddVersionKey "CompanyName" "Jiva Studio"
VIAddVersionKey "FileDescription" "Numen"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "Jiva Studio"

!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

; The window draws through Edge's engine, which Windows 11 and most of Windows
; 10 already carry. A machine without it is handed Microsoft's own installer.
Section "-WebView2"
  ReadRegStr $0 HKLM \
    "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  ${If} $0 == ""
    ReadRegStr $0 HKCU \
      "SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  ${EndIf}
  ${If} $0 == ""
    DetailPrint "Installing the WebView2 runtime"
    SetOutPath "$PLUGINSDIR"
    File "/oname=webview2.exe" "${WEBVIEW2}"
    ExecWait '"$PLUGINSDIR\webview2.exe" /silent /install' $1
    ${If} $1 != 0
      MessageBox MB_OK|MB_ICONEXCLAMATION \
        "The WebView2 runtime could not be installed. Numen will not open a window until it is."
    ${EndIf}
  ${EndIf}
SectionEnd

Section "Numen"
  SetOutPath "$INSTDIR"
  File "/oname=numen.exe" "${BINARY}"
  File "/oname=numen-flashcards.exe" "${FLASHCARDS}"
  File "/oname=numen-flashcards.ico" "${FLASHCARDS_ICON}"

  WriteRegStr HKLM "Software\Numen" "InstallDir" "$INSTDIR"
  WriteUninstaller "$INSTDIR\uninstall.exe"

  CreateShortcut "$SMPROGRAMS\Numen.lnk" "$INSTDIR\numen.exe"
  CreateShortcut "$SMPROGRAMS\Numen Flashcards.lnk" "$INSTDIR\numen-flashcards.exe" \
    "" "$INSTDIR\numen-flashcards.ico"

  !define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\Numen"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayName" "Numen"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "Publisher" "Jiva Studio"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "${UNINSTALL_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoRepair" 1
SectionEnd

; A vault is a folder of the person's own and is left where it is.
Section "Uninstall"
  Delete "$INSTDIR\numen.exe"
  Delete "$INSTDIR\numen-flashcards.exe"
  Delete "$INSTDIR\numen-flashcards.ico"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\Numen.lnk"
  Delete "$SMPROGRAMS\Numen Flashcards.lnk"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Numen"
  DeleteRegKey HKLM "Software\Numen"
SectionEnd
