@Echo Off
ChCp 65001 > Nul

Title Bottom child

Echo Hello from the bottom child

:Back
Echo %Time%
TimeOut /T 1 > Nul
GoTo Back

Pause
Exit /B 0
