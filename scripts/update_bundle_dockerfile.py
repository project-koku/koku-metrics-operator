#!/usr/bin/env python3

# this script strips "bundle/" from the COPY lines of the bundle.Dockerfile

input_file = "bundle.Dockerfile"

with open(input_file, 'r') as f:
    lines = f.readlines()

updated_lines = []
for line in lines:
    if line.startswith("COPY"):
        updated_lines.append(line.replace("bundle/", ""))
    else:
        updated_lines.append(line)

with open(input_file, 'w') as f:
    f.writelines(updated_lines)

print(f"Updated {input_file}")
