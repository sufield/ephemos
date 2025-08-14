#!/usr/bin/env python3
"""
Script to update Go dependency checksums in deps.bzl from go.sum
"""
import re
import sys

def parse_go_sum(go_sum_path):
    """Parse go.sum file and extract module checksums"""
    checksums = {}
    with open(go_sum_path, 'r') as f:
        for line in f:
            line = line.strip()
            if not line or '/go.mod' in line:
                continue
            parts = line.split()
            if len(parts) >= 3:
                module = parts[0]
                version = parts[1]
                checksum = parts[2]
                checksums[f"{module}@{version}"] = checksum
    return checksums

def update_deps_bzl(deps_bzl_path, checksums):
    """Update deps.bzl file with correct checksums"""
    with open(deps_bzl_path, 'r') as f:
        content = f.read()
    
    # Pattern to match go_repository entries
    pattern = r'go_repository\(\s*name\s*=\s*"([^"]+)",\s*importpath\s*=\s*"([^"]+)",\s*sum\s*=\s*"([^"]+)",\s*version\s*=\s*"([^"]+)",\s*\)'
    
    def replace_checksum(match):
        name = match.group(1)
        importpath = match.group(2)
        old_sum = match.group(3)
        version = match.group(4)
        
        key = f"{importpath}@{version}"
        if key in checksums:
            new_sum = checksums[key]
            print(f"Updating {importpath} {version}: {old_sum} -> {new_sum}")
            return f'go_repository(\n        name = "{name}",\n        importpath = "{importpath}",\n        sum = "{new_sum}",\n        version = "{version}",\n    )'
        else:
            print(f"Warning: No checksum found for {key}")
            return match.group(0)
    
    updated_content = re.sub(pattern, replace_checksum, content, flags=re.MULTILINE | re.DOTALL)
    
    with open(deps_bzl_path, 'w') as f:
        f.write(updated_content)
    
    print("deps.bzl updated successfully")

if __name__ == "__main__":
    checksums = parse_go_sum("go.sum")
    update_deps_bzl("deps.bzl", checksums)