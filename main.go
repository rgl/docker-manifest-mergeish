package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	spec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/regclient/regclient/regclient"
	"github.com/regclient/regclient/regclient/manifest"
	"github.com/regclient/regclient/regclient/types"
	"github.com/sirupsen/logrus"
)

type DockerManifestList struct {
	MediaType string `json:"mediaType"`
	spec.Index
}

type DockerImage struct {
	spec.Image
	OSVersion string `json:"os.version"`
}

func main() {
	log := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.WarnLevel,
	}

	debug := flag.Bool("debug", false, "enable debug log")
	targetManifestName := flag.String("target", "", "upload the merged manifest to this image manifest (without this, the manifest is written to stdout)")
	flag.Parse()

	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "Syntax: %s -target <target manifest name> <source manifest or image name>+\n", flag.CommandLine.Name())
		fmt.Fprintf(os.Stderr, "Example: %s -target ruilopes/test:v1.0.0 ruilopes/test:staging--v1.0.0-linux ruilopes/test:staging--v1.0.0-windows\n", flag.CommandLine.Name())
		os.Exit(1)
	}

	if *debug {
		log.Level = logrus.DebugLevel
	}

	ctx := context.Background()

	c := regclient.NewRegClient(
		regclient.WithLog(log),
		regclient.WithDockerCreds(),
		regclient.WithDockerCerts())

	imageIndex := DockerManifestList{}

	for _, name := range flag.Args() {
		ref, err := types.NewRef(name)
		if err != nil {
			log.Fatal(err)
		}

		manifest, err := c.ManifestGet(ctx, ref)
		if err != nil {
			log.Fatal(err)
		}

		switch manifest.GetMediaType() {
		case regclient.MediaTypeDocker2ManifestList: // application/vnd.docker.distribution.manifest.list.v2+json
			mergeManifestList(ctx, log, c, &imageIndex, ref, manifest)
		case regclient.MediaTypeDocker2Manifest: // application/vnd.docker.distribution.manifest.v2+json
			mergeManifest(ctx, log, c, &imageIndex, ref, manifest)
		default:
			log.Fatalf("Unsupported manifest: %s", manifest.GetMediaType())
		}
	}

	rawManifest, err := json.Marshal(imageIndex)
	if err != nil {
		log.Fatal(err)
	}

	if *targetManifestName != "" {
		targetManifestRef, err := types.NewRef(*targetManifestName)
		if err != nil {
			log.Fatal(err)
		}
		manifest, err := manifest.New(imageIndex.MediaType, rawManifest, targetManifestRef, nil)
		if err != nil {
			log.Fatal(err)
		}
		err = c.ManifestPut(ctx, targetManifestRef, manifest)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println(string(rawManifest))
	}
}

func mergeManifestList(ctx context.Context, log *logrus.Logger, c regclient.RegClient, imageIndex *DockerManifestList, ref types.Ref, manifest manifest.Manifest) {
	imageIndex.SchemaVersion = 2
	imageIndex.MediaType = regclient.MediaTypeDocker2ManifestList // application/vnd.docker.distribution.manifest.list.v2+json
	manifests, err := manifest.GetDescriptorList()
	if err != nil {
		log.Fatal(err)
	}
	for _, dockerImage := range manifests {
		log.WithFields(logrus.Fields{
			"ref":          ref.CommonName(),
			"architecture": dockerImage.Platform.Architecture,
			"os":           dockerImage.Platform.OS,
			"os.version":   dockerImage.Platform.OSVersion,
		}).Info("Found docker image")
	}
	imageIndex.Manifests = append(imageIndex.Manifests, manifests...)
}

func mergeManifest(ctx context.Context, log *logrus.Logger, c regclient.RegClient, imageIndex *DockerManifestList, ref types.Ref, manifest manifest.Manifest) {
	configDescriptor, err := manifest.GetConfigDescriptor()
	if err != nil {
		log.Fatal(err)
	}
	if configDescriptor.MediaType != regclient.MediaTypeDocker2ImageConfig { // "application/vnd.docker.container.image.v1+json"
		log.Fatalf("%s", configDescriptor.MediaType)
	}
	platform := configDescriptor.Platform
	if platform == nil {
		// NB c.BlobGetOCIConfig returned object does not contain "os.version", so we have to unmarshal ourselfs.
		dockerImageBlob, err := c.BlobGet(ctx, ref, configDescriptor.Digest, nil)
		if err != nil {
			log.Fatal(err)
		}

		rawBody, err := dockerImageBlob.RawBody()
		if err != nil {
			log.Fatal(err)
		}

		var dockerImage DockerImage

		if err := json.Unmarshal(rawBody, &dockerImage); err != nil {
			logrus.Fatal(err)
		}

		log.WithFields(logrus.Fields{
			"ref":          ref.CommonName(),
			"architecture": dockerImage.Architecture,
			"os":           dockerImage.OS,
			"os.version":   dockerImage.OSVersion,
		}).Info("Found docker image")

		platform = &spec.Platform{
			Architecture: dockerImage.Architecture,
			OS:           dockerImage.OS,
			OSVersion:    dockerImage.OSVersion,
		}
	}

	rawBody, err := manifest.RawBody()
	if err != nil {
		log.Fatal(err)
	}

	imageIndex.Manifests = append(imageIndex.Manifests, spec.Descriptor{
		MediaType: manifest.GetMediaType(),
		Digest:    manifest.GetDigest(),
		Platform:  platform,
		Size:      int64(len(rawBody)),
	})
}
