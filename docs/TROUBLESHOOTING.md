# AnyClaw 故障排查指南

本文档只列出当前 CLI 和 Gateway 已实现的排查路径，避免使用尚不存在的命令。

## 快速诊断

```bash
./anyclaw doctor --config anyclaw.json
./anyclaw config validate --config anyclaw.json
./anyclaw health --config anyclaw.json --verbose
./anyclaw status --config anyclaw.json --all
```

## Gateway 启动失败

### 检查端口

```bash
# Windows
netstat -ano | findstr 18789

# Linux/macOS
lsof -i :18789
```

### 检查配置

```bash
./anyclaw config validate --config anyclaw.json
./anyclaw config get gateway.host --config anyclaw.json
./anyclaw config get gateway.port --config anyclaw.json
```

### 前台启动观察错误

```bash
./anyclaw gateway run --config anyclaw.json
```

如果使用守护进程：

```bash
./anyclaw gateway daemon start --config anyclaw.json
./anyclaw gateway daemon stop --config anyclaw.json
```

## LLM 连接失败

### 检查当前模型配置

```bash
./anyclaw models status --config anyclaw.json
./anyclaw models list --config anyclaw.json
./anyclaw config get llm.provider --config anyclaw.json
./anyclaw config get llm.model --config anyclaw.json
```

### 常见原因

- API key 没有通过配置或环境变量提供。
- `llm.base_url` 指向了错误的兼容 API 地址。
- 模型名称不在服务商账号或本地 Ollama 中可用。
- 网络代理、防火墙或 DNS 阻止了出站请求。

## 渠道连接失败

```bash
./anyclaw channels list --config anyclaw.json
./anyclaw channels status --config anyclaw.json
```

如果 Gateway 未运行，`channels list` 会展示本地配置；`channels status` 会尝试访问 Gateway 并返回连接错误。

## 技能加载失败

```bash
./anyclaw skill list
./anyclaw skill info <skill-name>
```

检查项：

- `skills.dir` 或 `ANYCLAW_SKILLS_DIR` 是否指向正确目录。
- `skill.json` 是否为合法 JSON。
- 声明式技能是否包含 `prompts.system`，或安装流程是否从 `SKILL.md` 生成了提示词。

## 配对和移动端接入问题

配对命令需要 Gateway 正在运行：

```bash
./anyclaw gateway run --config anyclaw.json
./anyclaw pairing status --config anyclaw.json
./anyclaw pairing generate --config anyclaw.json --name "CLI Device" --type cli
./anyclaw pairing list --config anyclaw.json
```

## 性能问题

先确认 Gateway 的运行状态：

```bash
curl http://127.0.0.1:18789/healthz
curl http://127.0.0.1:18789/status
curl http://127.0.0.1:18789/runtimes
```

常见缓解方式：

- 降低 `llm.max_tokens`。
- 降低并发任务数量。
- 调整 `gateway.runtime_max_instances` 和 `gateway.runtime_idle_seconds`。
- 检查 `.anyclaw/audit/audit.jsonl` 是否快速增长。

## 审计日志

默认审计日志位置：

```text
.anyclaw/audit/audit.jsonl
```

查看方式：

```bash
# Windows PowerShell
Get-Content .anyclaw/audit/audit.jsonl -Tail 50

# Linux/macOS
tail -n 50 .anyclaw/audit/audit.jsonl
```

## 数据备份

AnyClaw 当前没有独立 `backup` CLI。建议直接备份配置、工作区和运行状态目录：

```bash
cp anyclaw.json anyclaw.backup.json
tar -czf anyclaw-data.tar.gz .anyclaw workflows skills plugins
```

## 报告问题

请在 GitHub Issue 中包含：

- 操作系统和版本
- AnyClaw commit 或版本
- 触发问题的命令
- 完整错误输出
- 脱敏后的配置片段
- 相关 `.anyclaw/audit/audit.jsonl` 行

## 下一步

- 阅读 [安全配置指南](SECURITY.md)
- 阅读 [部署指南](DEPLOYMENT.md)
