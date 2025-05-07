; OMSAY Chat Client Installer Script
[Setup]
AppName=OMSAY
AppVersion=25.5.7.4
DefaultDirName={commonpf}\OMSAY
DefaultGroupName=OMSAY
PrivilegesRequired=admin
UninstallDisplayIcon={app}\omsay.exe
OutputBaseFilename=OMSAY
Compression=lzma
SolidCompression=yes
DisableProgramGroupPage=yes
SetupIconFile=C:\omsay_build\omsay.ico

[Files]
Source: "C:\omsay_build\omsay.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "C:\omsay_build\omsay-updater.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "C:\omsay_build\omsay.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\OMSAY Chat"; Filename: "{app}\omsay.exe"; IconFilename: "{app}\omsay.ico"
Name: "{commondesktop}\OMSAY Chat"; Filename: "{app}\omsay.exe"; WorkingDir: "{app}"; IconFilename: "{app}\omsay.ico"
Name: "{commonstartup}\OMSAY Chat"; Filename: "{app}\omsay.exe"; IconFilename: "{app}\omsay.ico"

[Run]
Filename: "{app}\omsay.exe"; Description: "Launch OMSAY Chat"; Flags: nowait postinstall skipifsilent
