@Echo Off
ChCp 65001 >Nul

Cls

:: Build internal dependencies
PushD "..\internal\proxy"
Call ".\build.cmd"
PopD

Echo Done. Press any key to exit...
Pause >Nul
Exit /B 0
