// A generated module for Docker functions
//
// 该模块提供了一套简化的 Docker 镜像构建和推送工具。
// 它可以动态计算构建上下文、智能拼接镜像名称，并支持完整的构建参数透传。

package main

import (
	"context"
	"fmt"
	"strings"
	"dagger/docker/internal/dagger"
)

// Docker 提供了构建和管理 Docker 镜像的核心能力。
type Docker struct{}

// Build 执行 Docker 镜像的构建，并根据参数选择同步到缓存或推送到远程仓库。
//
// 示例用法:
// dagger call build --src . --registry docker.io/myuser --repo my-app --push true
func (m *Docker) Build(
	ctx context.Context,
	
	// src 是代码仓库的根目录，作为初始的构建来源
	src *dagger.Directory,
	
	// baseDir 是构建上下文的相对子目录路径。如果不为空，将以此目录作为 Docker 构建上下文
	// +optional
	baseDir string,
	
	// registry 是远程镜像仓库的地址 (例如: ghcr.io 或 docker.io/username)
	// +optional
	registry string,
	
	// repo 是镜像的名称。如果为空，将根据 baseDir 自动推导 (默认为 "app")
	// +optional
	repo string,
	
	// tag 是镜像的版本标签 (默认: "latest")
	// +optional
	// +default="latest"
	tag string,
	
	// push 决定是否在构建完成后将镜像推送到远程仓库。如果为 false，则仅执行构建并保留在 Dagger 缓存中
	// +optional
	push bool,
	
	// dockerfile 指定相对于构建上下文的 Dockerfile 路径 (默认: "Dockerfile")
	// +optional
	// +default="Dockerfile"
	dockerfile string,
	
	// target 指定多阶段构建 (Multi-stage build) 中的目标构建阶段名称
	// +optional
	target string,
	
	// platform 指定目标构建的平台架构 (例如: "linux/amd64", "linux/arm64")
	// +optional
	platform dagger.Platform,
	
	// buildArgs 传递给构建过程的参数列表，每项格式必须为 "KEY=VALUE"
	// +optional
	buildArgs []string,
	
	// secrets 传递给构建过程的安全凭证，用于在构建时安全地拉取私有依赖
	// +optional
	secrets []*dagger.Secret,
) (string, error) {

	// 1. 动态计算构建上下文与默认镜像名
	cleanDir := strings.Trim(baseDir, "./")
	var buildContext *dagger.Directory
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
	opts := dagger.DirectoryDockerBuildOpts{
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
		var parsedArgs []dagger.BuildArg
		for _, arg := range buildArgs {
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				parsedArgs = append(parsedArgs, dagger.BuildArg{
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