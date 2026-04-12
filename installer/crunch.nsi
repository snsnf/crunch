!include "MUI2.nsh"

; General
Name "Crunch"
OutFile "..\artifacts\Crunch-Setup-windows-amd64.exe"
InstallDir "$PROGRAMFILES\Crunch"
InstallDirRegKey HKLM "Software\Crunch" "InstallDir"
RequestExecutionLevel admin

; UI
!define MUI_ICON "..\gui\build\windows\icon.ico"
!define MUI_UNICON "..\gui\build\windows\icon.ico"
!define MUI_ABORTWARNING

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Language
!insertmacro MUI_LANGUAGE "English"

; Install section
Section "Install"
    SetOutPath "$INSTDIR"

    ; App files
    File "..\gui\build\bin\Crunch.exe"
    File "..\gui\build\bin\ffmpeg.exe"
    File "..\gui\build\bin\ffprobe.exe"

    ; Start Menu shortcut
    CreateDirectory "$SMPROGRAMS\Crunch"
    CreateShortcut "$SMPROGRAMS\Crunch\Crunch.lnk" "$INSTDIR\Crunch.exe"
    CreateShortcut "$SMPROGRAMS\Crunch\Uninstall.lnk" "$INSTDIR\Uninstall.exe"

    ; Desktop shortcut
    CreateShortcut "$DESKTOP\Crunch.lnk" "$INSTDIR\Crunch.exe"

    ; Uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Registry for Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crunch" "DisplayName" "Crunch"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crunch" "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crunch" "DisplayIcon" "$INSTDIR\Crunch.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crunch" "Publisher" "Crunch"
    WriteRegStr HKLM "Software\Crunch" "InstallDir" "$INSTDIR"
SectionEnd

; Uninstall section
Section "Uninstall"
    Delete "$INSTDIR\Crunch.exe"
    Delete "$INSTDIR\ffmpeg.exe"
    Delete "$INSTDIR\ffprobe.exe"
    Delete "$INSTDIR\Uninstall.exe"

    Delete "$SMPROGRAMS\Crunch\Crunch.lnk"
    Delete "$SMPROGRAMS\Crunch\Uninstall.lnk"
    RMDir "$SMPROGRAMS\Crunch"

    Delete "$DESKTOP\Crunch.lnk"

    RMDir "$INSTDIR"

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Crunch"
    DeleteRegKey HKLM "Software\Crunch"
SectionEnd
