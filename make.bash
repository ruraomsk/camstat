#!/bin/bash
echo 'Compiling'
go build
if [ $? -ne 0 ]; then
	echo 'An error has occurred! Aborting the script execution...'
	exit 1
fi
echo 'Copy camstat to device'
scp camstat rura@192.168.115.27:/home/rura
#scp test.bin admin@192.168.115.29:/home/admin