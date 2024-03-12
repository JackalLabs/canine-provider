#!/bin/bash

jprovd start --home="$HOME/providers/provider$1" -y --port "333$1" --moniker="provider$1" --threads=1 --interval=5
