@Echo Off
ChCp 65001 >Nul

:Back
Echo %Time%
TimeOut /T 1 >Nul
GoTo Back

Pause
Exit /B 0
