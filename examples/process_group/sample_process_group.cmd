@Echo Off
ChCp 65001 >Nul

Start "" /B /Wait "Netstat" "1"

Pause
Exit /B 0
