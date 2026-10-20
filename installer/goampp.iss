; GoAMPP installer (Inno Setup 6).
;
; Build from the repo root after goampp.exe exists:
;   iscc /DAppVersion=0.6.4 installer\goampp.iss
; Output lands in dist\goampp-setup-<version>.exe.
;
; Installs to C:\goampp: the short path matches the XAMPP / Laragon
; convention and keeps deep project paths under MAX_PATH. Writing to the
; drive root needs admin once; the install dir is then made writable for
; users so goampp itself runs unelevated (it writes config.json, bin\,
; downloads\, conf\, www\ ... next to the exe).

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

[Setup]
AppId={{6F1B7C1E-3A7D-4C55-9E0B-2D8A6C4F9B21}
AppName=GoAMPP
AppVersion={#AppVersion}
AppVerName=GoAMPP {#AppVersion}
AppPublisher=imtaqin
AppPublisherURL=https://github.com/imtaqin/goampp
AppSupportURL=https://github.com/imtaqin/goampp/issues
AppUpdatesURL=https://github.com/imtaqin/goampp/releases
VersionInfoVersion={#AppVersion}
SourceDir=..
OutputDir=dist
OutputBaseFilename=goampp-setup-{#AppVersion}
DefaultDirName={sd}\goampp
DefaultGroupName=GoAMPP
DisableProgramGroupPage=yes
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
SetupIconFile=assets\logo.ico
UninstallDisplayIcon={app}\goampp.exe
WizardStyle=modern
Compression=lzma2/max
SolidCompression=yes
CloseApplications=yes

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"

[Dirs]
; The whole tree must be user-writable: config.json lives in the root.
Name: "{app}"; Permissions: users-modify
Name: "{app}\bin"; Permissions: users-modify
Name: "{app}\downloads"; Permissions: users-modify
Name: "{app}\tmp"; Permissions: users-modify
Name: "{app}\logs"; Permissions: users-modify
Name: "{app}\conf"; Permissions: users-modify
Name: "{app}\www"; Permissions: users-modify

[Files]
; The icon, logos and manifest are compiled into goampp.exe.
Source: "goampp.exe"; DestDir: "{app}"; Flags: ignoreversion
; VC++ runtime seeded into PostgreSQL's bin\ by its post-install hook.
Source: "runtime\*.dll"; DestDir: "{app}\runtime"; Flags: ignoreversion

[InstallDelete]
; Leftovers from releases that shipped these next to the exe.
Type: files; Name: "{app}\goampp.exe.manifest"
Type: files; Name: "{app}\logo.ico"
Type: files; Name: "{app}\logo.png"
Type: filesandordirs; Name: "{app}\assets"

[Icons]
Name: "{group}\GoAMPP"; Filename: "{app}\goampp.exe"
Name: "{group}\{cm:UninstallProgram,GoAMPP}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\GoAMPP"; Filename: "{app}\goampp.exe"; Tasks: desktopicon

[Run]
; runasoriginaluser: don't leave the panel running elevated after setup.
Filename: "{app}\goampp.exe"; Description: "{cm:LaunchProgram,GoAMPP}"; Flags: nowait postinstall skipifsilent runasoriginaluser

[UninstallDelete]
; Caches only. bin\, conf\ and www\ hold user data and are left in place.
Type: filesandordirs; Name: "{app}\downloads"
Type: filesandordirs; Name: "{app}\tmp"
