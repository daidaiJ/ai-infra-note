# KServe + Knative 推理部署

> KServe 简化模型服务部署 + Knative 提供自动缩放和流量路由，构建完整的云原生 AI 推理平台。

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **完整学习文档** - KServe/Knative 概念、架构、业务场景实战 |
| [manifests/01-kserve-inferenceservice.yaml](manifests/01-kserve-inferenceservice.yaml) | 基础模型部署示例 |
| [manifests/02-knative-configuration.yaml](manifests/02-knative-configuration.yaml) | Knative 服务配置 |
| [manifests/03-traffic-splitting.yaml](manifests/03-traffic-splitting.yaml) | 流量分割示例 |
| [manifests/04-canary-deployment.yaml](manifests/04-canary-deployment.yaml) | Canary 发布完整示例 |

## 核心特性

```
┌─────────────────────────────────────────────┐
│         KServe + Knative 核心能力            │
├─────────────────────────────────────────────┤
│  ✅ 模型服务抽象  - 多框架统一部署接口        │
│  ✅ 自动扩缩容  - Scale-to-Zero 按需伸缩     │
│  ✅ 流量路由  - Canary/A/B 测试灰度发布      │
│  ✅ 版本管理  - Revision 控制快速回滚         │
└─────────────────────────────────────────────┘
```

## 项目结构

```
week8-kserve-knative/
├── README.md                              # 概览层文档
├── docs/README.md                         # 详解层文档
└── manifests/
    ├── 01-kserve-inferenceservice.yaml    # 基础模型部署
    ├── 02-knative-configuration.yaml      # Knative配置
    ├── 03-traffic-splitting.yaml          # 流量分割
    └── 04-canary-deployment.yaml          # Canary发布
```

## 使用示例

```yaml
# ============================================================
# 示例: 最小化 InferenceService 部署
# ============================================================
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: my-model
spec:
  predictor:
    pytorch:
      storageUri: gs://my-bucket/model-path
      resources:
        requests:
          cpu: "1"
          memory: 2Gi
        limits:
          nvidia.com/gpu: "1"
```

详见 **[docs/README.md](docs/README.md)** 获取 KServe + Knative 组合部署详解。
