// A generated module for Docker functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"fmt"
	"strings"
)

type Docker struct{}

func (m *Docker) Build(
	ctx context.Context,
	src *Directory,
	// +optional
	// +default=""
	baseDir string,
	// +optional
	// +default=""
	registry string,
	// +optional
	// +default=""
	repo string,
	// +optional
	// +default="latest"
	tag string,
	// +optional
	// +default=false
	push bool,
	// +optional
	// +default="Dockerfile"
	dockerfile string,
	// +optional
	// +default=""
	target string,
	// +optional
	platform Platform,
	// +optional
	buildArgs []string,
	// +optional
	secrets []*Secret,
) (string, error) {
	
	// 1. 动态计算构建上下文与默认镜像名
	cleanDir := strings.Trim(baseDir, "./")
	var buildContext *Directory
	var defaultRepo string

	if cleanDir != "" {
		buildContext = src.Directory(baseDir)
		defaultRepo = strings.ReplaceAll(cleanDir, "\\", "/")
	} else {
		buildContext = src
		defaultRepo = "app"
	}

	finalRepo := repo
	if finalRepo == "" {
		finalRepo = defaultRepo
	}

	// 2. 智能拼接目标地址
	var targetAddress string
	if registry != "" {
		targetAddress = fmt.Sprintf("%s/%s:%s", strings.TrimRight(registry, "/"), finalRepo, tag)
	} else {
		targetAddress = fmt.Sprintf("%s:%s", finalRepo, tag)
	}

	// 3. 动态拼装构建参数
	// Go SDK 中使用 Opts 结构体来替代 Python 的 kwargs
	opts := DirectoryDockerBuildOpts{
		Dockerfile: dockerfile,
	}
	
	if target != "" {
		opts.Target = target
	}
	
	if platform != "" {
		opts.Platform = platform
	}
	
	if len(secrets) > 0 {
		opts.Secrets = secrets
	}
	
	if len(buildArgs) > 0 {
		var parsedArgs []BuildArg
		for _, arg := range buildArgs {
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				parsedArgs = append(parsedArgs, BuildArg{
					Name:  parts[0],
					Value: parts[1],
				})
			}
		}
		opts.BuildArgs = parsedArgs
	}

	// 4. 执行构建
	builtContainer := buildContext.DockerBuild(opts)

	// 5. 执行推送或同步
	if push {
		_, err := builtContainer.Publish(ctx, targetAddress)
		if err != nil {
			return "", fmt.Errorf("推送失败: %w", err)
		}
		return fmt.Sprintf("已成功推送 %s", targetAddress), nil
	}

	_, err := builtContainer.Sync(ctx)
	if err != nil {
		return "", fmt.Errorf("构建失败: %w", err)
	}
	return fmt.Sprintf("已成功构建 %s", targetAddress), nil
}

/* // Returns a container that echoes whatever string argument is provided
func (m *Docker) ContainerEcho(stringArg string) *dagger.Container {
	return dag.Container().From("alpine:latest").WithExec([]string{"echo", stringArg})
}

// Returns lines that match a pattern in the files of the provided Directory
func (m *Docker) GrepDir(ctx context.Context, directoryArg *dagger.Directory, pattern string) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grep", "-R", pattern, "."}).
		Stdout(ctx)
}
 */