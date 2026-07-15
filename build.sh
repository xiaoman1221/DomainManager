#!/bin/bash
set -e

echo "Building frontend..."
cd web
npm install
npm run build
cd ..

echo "Building backend..."
go build -o domain-manager .

echo "Done: domain-manager"
