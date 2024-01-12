#!/usr/bin/env python3
import argparse
import re
from datetime import datetime, timezone
from tempfile import mkstemp
from shutil import move, copymode
from os import fdopen, remove

try:
    from packaging.version import parse
except ImportError as e:
    raise ImportError('`packaging` module not found, install with `pip3 install packaging`') from e

valid_semver = re.compile("^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$")

def check_version(new, old):
    if new == old:
        print("\nexpect new and previous versions to differ:\n\tnew version: %s\n\told version:" % new, old)
        exit(1)

    if (matched_new := re.fullmatch(valid_semver, new)) and (matched_old := re.fullmatch(valid_semver, old)):
        if parse(new) <= parse(old):
            print("\nnew version must sequentially follow old version!")
            exit(1)
        return

    print("\ninvalid version formats:")
    if not matched_new:
        print("\texpect new version format: X.Y.Z\n\tactual version format: %s" % new)
    if not matched_old:
        print("\texpect old version format: X.Y.Z\n\tactual version format: %s" % old)

    exit(1)

def replace(file_path, pattern, subst):
    fh, abs_path = mkstemp()
    with fdopen(fh,'w') as new_file:
        with open(file_path) as old_file:
            for line in old_file:
                new_file.write(line.replace(pattern, subst))
    copymode(file_path, abs_path)
    remove(file_path)
    move(abs_path, file_path)

def fix_csv(version, previous_version, image_sha, namespace):

    # get the operator description from docs
    docs = open("docs/csv-description.md")
    description = "    ".join(docs.readlines())

    # all the replacements that will be made in the CSV
    replacements = {
        "0001-01-01T00:00:00Z": f"{datetime.now(timezone.utc).replace(microsecond=0).isoformat()}Z",
        "INSERT-CONTAINER-IMAGE": f"{image_sha}",
        "INSERT-DESCRIPTION": f"|-\n    {description}",
        "name: Red Hat": f"name: Red Hat\n  replaces: koku-metrics-operator.v{previous_version}",
        "type: AllNamespaces": f"type: AllNamespaces\n  relatedImages:\n    - name: koku-metrics-operator\n      image: {image_sha}",
    }

    if namespace != "":
        replacements["namespace: placeholder"] = f"namespace: {namespace}"

    filename = f"koku-metrics-operator/{version}/manifests/koku-metrics-operator.clusterserviceversion.yaml"
    for k,v in replacements.items():
        replace(filename, k, v)

def fix_dockerfile(version):
    replacements = {
        "bundle/manifests": "manifests",
        "bundle/metadata": "metadata",
        "bundle/tests": "tests",
    }

    filename = f"koku-metrics-operator/{version}/Dockerfile"
    for k,v in replacements.items():
        replace(filename, k, v)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Script for updating the appropriate fields of the CSV")
    parser.add_argument("-n", "--namespace", help="namespace used for testing", default="")
    parser.add_argument("version", help="New version of the CSV")
    parser.add_argument("previous_version", help="Version of CSV being replaced")
    parser.add_argument("image_sha", help="The image sha of the compiled operator")
    args = parser.parse_args()
    print(vars(args))

    check_version(args.version, args.previous_version)

    fix_csv(args.version, args.previous_version, args.image_sha, args.namespace)
    fix_dockerfile(args.version)
