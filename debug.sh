#!/bin/bash

godebug build -instrument \
    git.coolaj86.com/coolaj86/go-telebitd/connection,git.coolaj86.com/coolaj86/go-telebitd/connection \
    -o debug .
