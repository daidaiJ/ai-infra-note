# vLLM + SGLang 推理引擎吞吐量优化

> 从推理引擎层面深入系统优化技术栈，覆盖 vLLM 和 SGLang 两大主流引擎的吞吐量优化策略。

---

## 快速导航

| 文件 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 📖 **完整学习文档入口** |
| [docs/01-common-optimization-strategy.md](docs/01-common-optimization-strategy.md) | 🧭 **通用优化思路指导** - 并发、TTFT、吞吐、低成本四个维度 |
| [docs/02-vllm-optimization.md](docs/02-vllm-optimization.md) | 🔵 **vLLM 专有优化策略** - PagedAttention/APC/Speculative Decoding |
| [docs/03-sglang-optimization.md](docs/03-sglang-optimization.md) | 🟠 **SGLang 专有优化策略** - RadixAttention/PD Disaggregation/EAGLE |
| [docs/04-comparison-and-selection.md](docs/04-comparison-and-selection.md) | ⚖️ **对比与选型** - 特性矩阵 + 场景选型决策树 |

## 核心特性

```
┌─────────────────────────────────────────────────────────────┐
│            vLLM + SGLang 推理优化知识体系                    │
├─────────────────────────────────────────────────────────────┤
│  ✅ 通用优化思路  - 并发/TTFT/吞吐/低成本四维指导           │
│  ✅ vLLM 专有优化  - PagedAttention/APC/推测解码/量化       │
│  ✅ SGLang 专有优化 - RadixAttention/PD分离/EAGLE/HiCache  │
│  ✅ 对比与选型    - 特性矩阵 + 场景决策树                   │
└─────────────────────────────────────────────────────────────┘
```

## 项目结构

```
week5-vllm-sglang-optimization/
├── README.md                                   # 本文件
└── docs/
    ├── README.md                               # 详解层入口
    ├── 01-common-optimization-strategy.md       # 通用优化思路指导
    ├── 02-vllm-optimization.md                 # vLLM 专有优化策略
    ├── 03-sglang-optimization.md               # SGLang 专有优化策略
    └── 04-comparison-and-selection.md          # 对比与选型
```

> 本专题为参考资料，推理引擎优化由部署优化团队负责，此处仅作为知识归档。

详见 **[docs/README.md](docs/README.md)** 获取完整学习文档。
