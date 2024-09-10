#!/bin/bash

if $1 == "0"
    # Move to export directory
    cd /export

    # Execute the ipong command
    ipoing -G -c 1 .

    # Check the exit status of the command
    if [ $? -eq 0 ]; then
        # Command succeeded
        echo "Able to read and write to the nfs mount path /export"
    else
        # Command failed
        exit 1
    fi
    #Logic for showmount
fi

if $1 == "1"
#####Temp Code -Remove later #########
    # Move to export directory
    cd /export

    # Execute the ipong command
    ipoing -G -c 1 .

    # Check the exit status of the command
    if [ $? -eq 0 ]; then
        # Command succeeded
        echo "Able to read and write to the nfs mount path /export"
    else
        # Command failed
        exit 1
    fi
#####Temp Code -Remove later #########
    #Logic for showmount
fi