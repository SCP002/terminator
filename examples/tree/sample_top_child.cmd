@Echo Off
ChCp 65001 > Nul

Title Top child

Echo Hello from the top child

Start "" "sample_middle_child.cmd"

Pause
Exit /B 0
