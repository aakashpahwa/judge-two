#!/bin/bash

function default {
    echo "No default command"
}

function create {
    # Check for kubectl, helm, helm repo additions, docker etc
    echo "Building Project..."
    initDatabase
}

function initDatabase {
    echo "Creating database..."
    (helm install db ./db --namespace=judge --create-namespace && echo -e "Database deployed.\nBe patient while the containers are being created.") || echo "Database creation failed."
}

${@:-default}