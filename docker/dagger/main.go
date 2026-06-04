// A generated module for Docker functions
//
// 该模块提供了一套简化的 Docker 镜像构建和推送工具。
// 它可以动态计算构建上下文、智能拼接镜像名称、支持完整的构建参数透传。

package main

import (
    "context"
    "fmt"
    "strings"

    "dagger/docker/internal/dagger"
)

// Docker 提供了构建和管理 Docker 镜像的核心能力。
type Docker struct{}

// Build 执行 Docker 镜像的构建，并根据配置决定是仅保留在本地缓存，还是推送到远程镜像仓库。
//
// 示例用法 (本地 Dagger CLI):
// dagger call build --src . --registry ghcr.io/myuser --repo my-app --tags "v1.0.0,latest" --push true
func (m *Docker) Build(
    ctx context.Context,
    
    // src 是代码仓库的根目录，作为初始的构建来源
    src *dagger.Directory,
    
    // baseDir 是构建上下文的相对子目录路径。如果不为空，将以此目录作为 Docker 的构建上下文 (Build Context)
    // +optional
    baseDir string,
    
    // registry 是远程镜像仓库的地址 (例如: ghcr.io 或 docker.io/username)
    // 如果为空，构建出的镜像将只包含 repo 名称
    // +optional
    registry string,
    
    // repo 是镜像的名称。如果为空，将智能根据 baseDir 目录路径自动推导 (默认 fallback 为 "app")
    // +optional
    repo string,
    
    // tags 接收纯文本字符串，支持以逗号 (,) 或换行符 (\n) 分隔的多个版本标签。
    // 如果为空，则默认赋予 "latest" 标签。
    // +optional
    tags string,
    
    // push 决定是否在构建完成后将镜像推送到远程仓库。
    // 如果为 false (默认)，则仅执行构建并将其保留在 Dagger 的高速缓存中，验证构建是否能成功。
    // +optional
    push bool,
    
    // dockerfile 指定相对于构建上下文的 Dockerfile 路径
    // +optional
    // +default="Dockerfile"
    dockerfile string,
    
    // target 指定多阶段构建 (Multi-stage build) 中需要构建到的目标阶段 (Stage) 名称
    // +optional
    target string,
    
    // platform 指定目标构建的操作系统和硬件架构 (例如: "linux/amd64", "linux/arm64")
    // 适用于跨平台交叉编译
    // +optional
    platform dagger.Platform,
    
    // buildArgs 传递给构建过程的参数列表 (--build-arg)，每项格式严格要求为 "KEY=VALUE"
    // +optional
    buildArgs []string,
    
    // secrets 传递给构建过程的安全凭证 (--secret)，用于在构建时安全地拉取私有依赖而不会泄露在镜像层中
    // +optional
    secrets []*dagger.Secret,
) (string, error) {

    // ==========================================
    // 0. 解析并清洗 Tags 字符串
    // ==========================================
    var tagList []string
    
    // 消除 Windows 换行符的干扰，并将所有换行符统一替换为逗号，方便后续的一维数组分割
    normalizedTags := strings.ReplaceAll(tags, "\r\n", ",")
    normalizedTags = strings.ReplaceAll(normalizedTags, "\n", ",")
    
    // 分割并去除空白字符，剔除空项
    for _, t := range strings.Split(normalizedTags, ",") {
        cleanTag := strings.TrimSpace(t)
        if cleanTag != "" {
            tagList = append(tagList, cleanTag)
        }
    }

    // 默认回退处理：如果未传入任何有效标签，默认使用 latest
    if len(tagList) == 0 {
        tagList = []string{"latest"}
    }

    // ==========================================
    // 1. 动态计算构建上下文 (Context) 与默认镜像名
    // ==========================================
    cleanDir := strings.Trim(baseDir, "./")
    var buildContext *dagger.Directory
    var defaultRepo string

    if cleanDir != "" {
        buildContext = src.Directory(baseDir)
        // 将嵌套目录 (如 backend/api) 转换为标准的镜像名格式 (backend/api)
        defaultRepo = strings.ReplaceAll(cleanDir, "\\", "/")
    } else {
        buildContext = src
        defaultRepo = "app"
    }

    finalRepo := repo
    if finalRepo == "" {
        finalRepo = defaultRepo
    }

    // ==========================================
    // 2. 智能拼接所有目标推送地址
    // ==========================================
    var targetAddresses []string
    for _, tag := range tagList {
        var addr string
        if registry != "" {
            // 确保 registry 结尾没有多余的斜杠，防止拼出双斜杠 (ghcr.io//app:tag)
            addr = fmt.Sprintf("%s/%s:%s", strings.TrimRight(registry, "/"), finalRepo, tag)
        } else {
            addr = fmt.Sprintf("%s:%s", finalRepo, tag)
        }
        targetAddresses = append(targetAddresses, addr)
    }

    // ==========================================
    // 3. 动态拼装构建参数 (Build Options)
    // ==========================================
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

    // 解析 "KEY=VALUE" 格式的构建参数
    if len(buildArgs) > 0 {
        var parsedArgs []dagger.BuildArg
        for _, arg := range buildArgs {
            if strings.Contains(arg, "=") {
                parts := strings.SplitN(arg, "=", 2)
                parsedArgs = append(parsedArgs, dagger.BuildArg{
                    Name:  strings.TrimSpace(parts[0]),
                    Value: strings.TrimSpace(parts[1]),
                })
            }
        }
        opts.BuildArgs = parsedArgs
    }

    // ==========================================
    // 4. 发起构建指令
    // ==========================================
    // 注意：这一步只是在 Dagger 引擎中定义了 DAG 节点，尚未发生真正的物理构建。
    builtContainer := buildContext.DockerBuild(opts)

    // ==========================================
    // 5. 执行推送 (Publish) 或 同步验证 (Sync)
    // ==========================================
    if push {
        // 当执行推送时，循环遍历所有的 Tag。
        // 💡 Dagger 引擎层具备高度缓存复用：
        // 第一个 tag 的 Publish 会触发真正的构建和图层(Layers)上传；
        // 后续 tag 的 Publish 瞬间完成，仅推送轻量级的 Manifest 关联信息。
        for _, addr := range targetAddresses {
            _, err := builtContainer.Publish(ctx, addr, dagger.ContainerPublishOpts{OciMediaTypes: true})
            if err != nil {
                return "", fmt.Errorf("❌ 镜像推送失败 [%s]: %w", addr, err)
            }
        }
        return fmt.Sprintf("✅ 镜像已成功构建并推送到: \n- %s", strings.Join(targetAddresses, "\n- ")), nil
    }

    // 如果未开启 push，则强制调用 Sync 触发实际构建过程并保留在缓存中。
    // 这对于 CI/CD 的 PR 检查阶段（Dry-Run）非常有用。
    _, err := builtContainer.Sync(ctx)
    if err != nil {
        return "", fmt.Errorf("❌ 镜像构建失败: %w", err)
    }
    return fmt.Sprintf("✅ 镜像已成功构建: \n- %s", strings.Join(targetAddresses, "\n- ")), nil
}