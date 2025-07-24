package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	corev1 "k8s.io/api/core/v1"
)

type ArmResult struct {
	Supported bool
	Err       error
}

const MaxConcurrent = 7

var armResultCache = make(map[string]ArmResult)
var armResultCacheMutex sync.RWMutex

func CheckAllWorkloadsArm(workloads []Workload) []ArmResult {
	results := make([]ArmResult, len(workloads))

	var wg sync.WaitGroup
	sem := make(chan struct{}, MaxConcurrent)

	for idx := range workloads {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			cacheKey := fmt.Sprintf("%s:%s:%s", workloads[i].Kind, workloads[i].Name, workloads[i].Namespace)

			armResultCacheMutex.RLock()
			cachedResult, found := armResultCache[cacheKey]
			armResultCacheMutex.RUnlock()

			if found {
				results[i] = cachedResult
				return
			}

			var supported bool
			var err error

			for {
				supported, err = CheckWorkloadSupportsArm(&workloads[i])
				if err != nil && strings.Contains(err.Error(), "TOOMANYREQUESTS") {
					time.Sleep(time.Millisecond * time.Duration(rand.Int63n(800)))
				} else {
					break
				}
			}
			result := ArmResult{Supported: supported, Err: err}
			results[i] = result

			armResultCacheMutex.Lock()
			armResultCache[cacheKey] = result
			armResultCacheMutex.Unlock()
		}(idx)
	}
	wg.Wait()
	return results
}

func CheckWorkloadSupportsArm(w *Workload) (bool, error) {
	var podSpec *corev1.PodSpec
	switch w.Kind {
	case WorkloadDeployment:
		podSpec = &w.deployment.Spec.Template.Spec
	case WorkloadStatefulSet:
		podSpec = &w.statefulSet.Spec.Template.Spec
	default:
		return false, fmt.Errorf("unsupported workload kind: %s", w.Kind)
	}
	images := getPodTemplateImages(*podSpec)

	supportArm := true
	for _, image := range images {
		ret, err := imageSupportsArm64(image)
		if err != nil {
			return false, fmt.Errorf("failed to check image %s for arm64 support: %w", image, err)
		}
		if ret == false {
			supportArm = false
			break
		}
	}
	return supportArm, nil
}

func getPodTemplateImages(podSpec corev1.PodSpec) []string {
	var images []string
	for _, c := range podSpec.Containers {
		images = append(images, c.Image)
	}
	for _, c := range podSpec.InitContainers {
		images = append(images, c.Image)
	}
	return images
}

var keychain = authn.NewMultiKeychain(
	authn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard))), // ECR
	authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper()),         // ACR
	google.Keychain,       // GCR & Artifact Registry
	github.Keychain,       // GHCR
	authn.DefaultKeychain, // local ~/.docker/config.json
)

// imageSupportsArm64 checks whether the given container image supports the linux/arm64
// platform. It returns true if the image manifest list (index) contains an arm64
// variant, or if the single-arch image itself is built for arm64.
func imageSupportsArm64(imageRef string) (bool, error) {
	// Parse an arbitrary image reference (registry/name:tag or digest).
	ref, err := name.ParseReference(imageRef, name.WeakValidation)
	if err != nil {
		return false, fmt.Errorf("failed to parse image reference: %w", err)
	}

	// Pull the descriptor (manifest or index) from the remote registry.
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(keychain),
		remote.WithContext(context.Background()),
	}
	desc, err := remote.Get(ref, remoteOpts...)
	if err != nil {
		return false, fmt.Errorf("failed to fetch image descriptor: %w", err)
	}

	mt := desc.Descriptor.MediaType
	// Handle multi-arch images (OCI index / Docker manifest list).
	if mt == types.OCIImageIndex || mt == types.DockerManifestList {
		idx, err := desc.ImageIndex()
		if err != nil {
			return false, fmt.Errorf("failed to load image index: %w", err)
		}
		indexManifest, err := idx.IndexManifest()
		if err != nil {
			return false, fmt.Errorf("failed to read index manifest: %w", err)
		}
		for _, manifest := range indexManifest.Manifests {
			plat := manifest.Platform
			if plat != nil && plat.Architecture == "arm64" && strings.EqualFold(plat.OS, "linux") {
				return true, nil // linux/arm64 variant found
			}
		}
		return false, nil // no arm64 variant in the index
	}

	// Handle single-arch images.
	img, err := desc.Image()
	if err != nil {
		return false, fmt.Errorf("failed to load image: %w", err)
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		return false, fmt.Errorf("failed to read image config: %w", err)
	}
	return cfg.Architecture == "arm64", nil
}
