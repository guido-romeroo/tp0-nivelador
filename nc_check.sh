#!/bin/bash

NETWORK="tp0-nivelador_default"
COMANDO='apk add --no-cache netcat-openbsd && echo "Hello World" | nc server 5678'
docker run --rm --network="$NETWORK" alpine sh -c "$COMANDO"