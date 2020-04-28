#!/bin/bash

godebug build -instrument \
    git.coolaj86.com/coolaj86/go-telebitd/rvpn/connection,git.coolaj86.com/coolaj86/go-telebitd/rvpn/connection \
    -o debug .
