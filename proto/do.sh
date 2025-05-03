#!/bin/bash

# Loop through all subdirectories in the current directory
for dir in */; do
  # Remove trailing slash from directory name
  dirname="${dir%/}"

  echo "Processing directory: $dirname"

  # Change into the subdirectory
  cd "$dir" || continue

  # Run the three commands
  python3 -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. *.proto
  protoc -I . *.proto --go_out=plugins=grpc:.
  find . -maxdepth 1 -type f ! -name '*.proto' -exec mv {} ../../internal/"$dirname"/ \;

  # Return to the parent directory
  cd ..

  echo "Finished $dirname"
  echo "----------------------"
done
