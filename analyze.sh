echo "datasize"
find ./*.bin | xargs -I {} -- ./analyze_datasize.sh {}
echo "dataheader"
find ./*.bin | xargs -I {} -- ./analyze_dataheader.sh {}
exit
echo "UDP"
find ./*.bin | xargs -I {} -- ./analyze_udp.sh {}
echo "terminator"
find ./*.bin | xargs -I {} -- ./analyze_terminator.sh {}
echo "line header"
find ./*.bin | xargs -I {} -- ./analyze_line_header.sh {}
echo "cmd: 1b7b05*"
find ./*.bin | xargs -I {} -- ./analyze_1b7b05.sh {}
echo "cmd: 1b7b07*"
find ./*.bin | xargs -I {} -- ./analyze_1b7b07.sh {}
