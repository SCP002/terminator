@Echo Off
ChCp 65001 >Nul

Cls

:: Build internal dependencies
PushD "..\internal\proxy"
Call ".\build.cmd"
PopD

Exit /B 0
