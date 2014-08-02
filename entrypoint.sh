#!/bin/sh

goose up && mizumanju -d=$DATABASE_URL -h=$LISTEN_IP -m=$MAIL_ADDRESS -n=$NAME -p=$LISTEN_PORT -pp=$DEBUG_SERVER -sh=$SMTP_HOST -sp=$SMTP_PORT -ss=$SMTP_START_TLS -su=$SMTP_USER -sw=$SMTP_PASSWORD -u=$BASE_URL
