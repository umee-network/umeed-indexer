#!/bin/bash -eux

# Runs the firebase emulator with the firebase config json.

CWD="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

FIREBASE_CONFIG_PATH="${FIREBASE_CONFIG_PATH:-$CWD/firebase.json}"
FIREBASE_PROJECT_ID="${FIREBASE_PROJECT_ID:-convexity-bonds}"


# TODO: create script to automate this.
# How to get a fresh cloud data https://firebase.google.com/docs/firestore/manage-data/export-import#gcloud_5
# https://stackoverflow.com/questions/57838764/how-to-import-data-from-cloud-firestore-to-the-local-emulator
IMPORT_DATA_PATH="${IMPORT_DATA_PATH:-$CWD/cloud-data/default}"

# TODO: create script to automate this.
# How to load users into the exported data
# https://stackoverflow.com/questions/73703261/is-it-possible-to-import-authentication-data-to-firebase-emulators

# Makes sure firebase emulators exist
if ! command -v firebase &> /dev/null
then
  echo "⚠️ firebase command could not be found!"
  echo "Running command to install"
  curl -sL firebase.tools | bash
fi

firebase -c $FIREBASE_CONFIG_PATH emulators:start --only firestore,auth --project $FIREBASE_PROJECT_ID --import $IMPORT_DATA_PATH --export-on-exit