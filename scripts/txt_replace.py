#!/usr/bin/env python3

import sys
from datetime import datetime
from tempfile import mkstemp
from shutil import move, copymode
from os import fdopen, name, path, remove


def check_version(v_tup):
    new, old, _ = v_tup
    if new == old:
        print("expect new and previous versions to differ:\n\tnew version: %s\n\told version:" % new, old)
        exit()
    split = new.split(".")
    if len(split) != 3:
        print("expect version format: X.Y.Z\nactual version format: %s" % v)
        exit()
    for value in split:
        try:
            int(value)
        except ValueError:
            print("expect version format: X.Y.Z\nactual version format: %s" % v)
            exit()

def replace(file_path, pattern, subst):
    fh, abs_path = mkstemp()
    with fdopen(fh,'w') as new_file:
        with open(file_path) as old_file:
            for line in old_file:
                new_file.write(line.replace(pattern, subst))
    copymode(file_path, abs_path)
    remove(file_path)
    move(abs_path, file_path)

def fix_csv(version_tuple):
    version, previous, sha = version_tuple
    # get the operator description from docs
    docs = open("docs/csv-description.md")
    description = "    ".join(docs.readlines())

    # all the replacements that will be made in the CSV
    replacements = {
        "0001-01-01T00:00:00Z": datetime.utcnow().replace(microsecond=0).isoformat() + "Z",
        "INSERT-CONTAINER-IMAGE": f"quay.io/project-koku/koku-metrics-operator:v{version}",
        "INSERT-DESCRIPTION": "|-\n    " + description,
        "name: Red Hat": f"name: Red Hat\n  replaces: koku-metrics-operator.v{previous}",
        "type: AllNamespaces": f"type: AllNamespaces\n  relatedImages:\n    - name: koku-metrics-operator\n      image: {sha}"
    }

    filename = f"koku-metrics-operator/{version}/manifests/koku-metrics-operator.clusterserviceversion.yaml"
    for k,v in replacements.items():
        replace(filename, k, v)

def fix_dockerfile(version_tuple):
    version, *_ = version_tuple
    replacements = {
        "bundle/manifests": "manifests",
        "bundle/metadata": "metadata",
    }

    filename = f"koku-metrics-operator/{version}/Dockerfile"
    for k,v in replacements.items():
        replace(filename, k, v)

if __name__ == "__main__":
    nargs = len(sys.argv)
    if nargs != 4:
        print("usage: %s VERSION PREVIOUS_VERSION IMAGE_SHA" % path.basename(sys.argv[0]))
        exit()

    version_tuple = sys.argv[1:]
    check_version(version_tuple)

    fix_csv(version_tuple)
    fix_dockerfile(version_tuple)
