@Echo Off
ChCp 65001 >Nul

Cls

:: Build internal dependencies
PushD "..\internal\kamikaze"
Call ".\build.cmd"
PopD

PushD "..\internal\send_message"
Call ".\build.cmd"
PopD

Exit /B 0
