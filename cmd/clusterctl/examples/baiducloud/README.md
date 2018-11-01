# Baidu Example Manifests

This directory contains example manifest files that can be used to create a fully-functional cluster.

## Generation

For convenience, a generation script which populates templates manifests is provided.

1. Run the generation script.
```bash
./generate-yaml.sh
```

If the yaml files already exist, you will see an error like the one below:
```bash
$ ./generate-yaml.sh
File provider-components.yaml already exists. Delete it manually before running this script.
```
