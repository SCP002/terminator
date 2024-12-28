@Echo Off
ChCp 65001 > Nul

Cls

Rem Build internal dependencies
PushD "..\internal\proxy"
Call ".\build.cmd"
PopD

Exit /B 0
