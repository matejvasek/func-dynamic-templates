package main

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"github.com/matejvasek/func-dynamic-tempates/lib"
	"io"
	fn "knative.dev/kn-plugin-func"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
		<-sigs
		os.Exit(1)
	}()

	rootCmd, err := lib.NewCommandFromRepository(quarkusRepository{})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	err = rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

type quarkusRepository struct{}

func (q quarkusRepository) Runtimes(context.Context) ([]string, error) {
	return []string{"quarkus"}, nil
}

func (q quarkusRepository) Templates(ctx context.Context, runtime string) ([]string, error) {
	if runtime != "quarkus" {
		return nil, fmt.Errorf("unknown runtime: %q", runtime)
	}
	return []string{"cloudevents", "http"}, nil
}

func (q quarkusRepository) Template(ctx context.Context, runtime, name string) (lib.Template, error) {
	return quarkusTemplate{name}, nil
}

type quarkusTemplate struct {
	name string
}

func (q quarkusTemplate) Name() string {
	return q.name
}

func (q quarkusTemplate) Runtime() string {
	return "quarkus"
}

func (q quarkusTemplate) Write(ctx context.Context, name, destDir string) error {
	var err error

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if destDir == "" {
		destDir = filepath.Join(wd, name)
	}

	runtime := q.Runtime()
	template := q.Name()

	group := "org.acme"
	artifact := name
	if artifact == "" {
		artifact = fmt.Sprintf("func-%s-%s", runtime, template)
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr, "What group name use (default: %s)?", group)
	text, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	g := strings.Trim(text, " \t\r\n")
	if g != "" {
		group = g
	}

	reader = bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr, "What artifact name use (default: %s)?", artifact)
	text, err = reader.ReadString('\n')
	if err != nil {
		return err
	}
	a := strings.Trim(text, " \t\r\n")
	if a != "" {
		artifact = a
	}

	var httpClient http.Client

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://code.quarkus.io/d", nil)
	if err != nil {
		return err
	}
	query := req.URL.Query()
	query.Add("g", group)
	query.Add("a", artifact)
	query.Add("cn", "code.quarkus.io")

	switch template {
	case "http":
		query.Add("e", "funqy-http")
	case "cloudevents":
		query.Add("e", "funqy-knative-events")
	default:
		return fmt.Errorf("unknown template: %q", template)
	}

	req.URL.RawQuery = query.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmpDir, err := os.MkdirTemp("", "template-temp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tmpZipFilename := filepath.Join(tmpDir, "app.zip")

	tmpZip, err := os.OpenFile(tmpZipFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer tmpZip.Close()

	_, err = io.Copy(tmpZip, resp.Body)
	if err != nil {
		return err
	}

	err = unzip(tmpZipFilename, destDir)
	if err != nil {
		return err
	}

	// write func.yaml
	return fn.Function{
		Name:     name,
		Root:     destDir,
		Runtime:  q.Runtime(),
		Template: q.Name(),
		Builder:  "default",
		Builders: map[string]string{
			"default": "quay.io/boson/faas-jvm-builder:v0.8.4",
			"jvm":     "quay.io/boson/faas-jvm-builder:v0.8.4",
			"native":  "quay.io/boson/faas-quarkus-native-builder:v0.8.4",
		},
		HealthEndpoints: fn.HealthEndpoints{
			Liveness: "/health/liveness",
			Readiness: "/health/readiness",
		},
	}.WriteConfig()
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		// remove first part of path
		var nameWithoutFirstPart string
		parts := strings.Split(f.Name, "/")
		if len(parts) >= 1 {
			nameWithoutFirstPart = filepath.Join(parts[1:]...)
		}


		if nameWithoutFirstPart == "" {
			return nil
		}
		path := filepath.Join(dest, nameWithoutFirstPart)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
