# About

[![Build status](https://img.shields.io/github/workflow/status/rgl/docker-manifest-mergeish/Build)](https://github.com/rgl/docker-manifest-mergeish/actions?query=workflow%3ABuild)
[![Docker pulls](https://img.shields.io/docker/pulls/ruilopes/docker-manifest-mergeish)](https://hub.docker.com/repository/docker/ruilopes/docker-manifest-mergeish)

This is a quick&dirty application to create a docker manifest by crude merging existing manifests and images.

This will also try to add the `platform` manifest property. This is required to support different Windows versions on the same manifest.

This is mainly used by the GitHub Actions workflow at [rgl/example-docker-buildx-go](https://github.com/rgl/example-docker-buildx-go).

**This does not do any validation, so make sure all the included manifests and images can be used together.**

**This only supports docker v2 manifests and images. There is no support for OCI.**

## Usage

Create a bash function to execute `docker-manifest-mergeish` from a container:

```bash
function docker-manifest-mergeish {
    docker container run --rm \
        -u "$(id -u):$(id -g)" -e HOME -v "$HOME:$HOME:ro" \
        -v /etc/docker/certs.d:/etc/docker/certs.d:ro \
        ruilopes/docker-manifest-mergeish:v0.0.1 "$@"
}
```

Try it:

```console
$ docker-manifest-mergeish -help
Usage of /app/docker-manifest-mergeish:
  -debug
        enable debug log
  -target string
        upload the merged manifest to this image manifest (without this, the manifest is written to stdout)
```

Use it alike:

```console
$ docker-manifest-mergeish \
    -debug \
    -target ruilopes/example-docker-buildx-go:v1.3.0-test123 \
    ruilopes/example-docker-buildx-go:staging--v1.3.0-test123-linux \
    ruilopes/example-docker-buildx-go:staging--v1.3.0-test123-windowsnanoserver-1809 \
    ruilopes/example-docker-buildx-go:staging--v1.3.0-test123-windowsnanoserver-ltsc2022
```

# Notes

* This is largely based on [regctl](https://github.com/regclient/regclient).
* You might also find [go-containerregistry](https://github.com/google/go-containerregistry/issues/1137) useful.
