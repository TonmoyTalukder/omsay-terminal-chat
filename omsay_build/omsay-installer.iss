; OMSAY Chat Client Installer Script
[Setup]
AppName=OMSAY Chat
AppVersion=25.5.6.4
DefaultDirName={pf}\OMSAY
DefaultGroupName=OMSAY Chat
UninstallDisplayIcon={app}\omsay.exe
OutputBaseFilename=OMSAY-Setup
Compression=lzma
SolidCompression=yes
DisableProgramGroupPage=yes

[Files]
Source: "C:\omsay_build\omsay.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\OMSAY Chat"; Filename: "{app}\omsay.exe"
Name: "{userdesktop}\OMSAY Chat"; Filename: "{app}\omsay.exe"; WorkingDir: "{app}"

[Run]
Filename: "{app}\omsay.exe"; Description: "Launch OMSAY Chat"; Flags: nowait postinstall skipifsilent

[Icons]
Name: "{userstartup}\OMSAY Chat"; Filename: "{app}\omsay.exe"
