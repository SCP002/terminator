@Echo Off
ChCp 65001 >Nul

Cls

:: Build internal dependencies
PushD "..\internal\proxy"
Call ".\build.cmd"
PopD

Echo Build done. Press any key to continue...
Pause >Nul
Exit /B 0
