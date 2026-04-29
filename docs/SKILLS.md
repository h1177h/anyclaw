# AnyClaw 技能系统

AnyClaw 技能通过 `skills/` 目录扩展 Agent 的提示词、工具说明和操作约束。当前 CLI 已实现技能搜索、安装、列表、详情和目录查询。

## 技能目录结构

```text
skills/
  my-skill/
    SKILL.md
    skill.json
    references/
      *.md
```

`skill.json` 提供元数据，`SKILL.md` 提供面向 Agent 的主要说明。声明式技能如果需要运行时直接读取系统提示，应在 `skill.json` 的 `prompts.system` 中放入核心提示，或者确保安装流程会从 `SKILL.md` 转换。

## skill.json 示例

```json
{
  "name": "my-skill",
  "description": "Describe when and how this skill should be used.",
  "version": "1.0.0",
  "source": "local",
  "tags": ["productivity"],
  "permissions": ["read", "write"],
  "prompts": {
    "system": "Use this skill when..."
  }
}
```

## 已支持的 CLI

```bash
# 搜索远端技能
./anyclaw skill search github

# 安装内置或远端技能
./anyclaw skill install <name>

# 从 GitHub 仓库路径安装
./anyclaw skill install <owner>/<repo>/<skill>

# 列出本地技能
./anyclaw skill list

# 查看本地技能详情
./anyclaw skill info <name>

# 查询技能目录
./anyclaw skill catalog [query]
```

`skills` 是 `skill` 的兼容别名，因此 `./anyclaw skills list` 也可以使用。

## 配置技能目录

默认技能目录是 `skills`。可以通过环境变量覆盖：

```bash
ANYCLAW_SKILLS_DIR=/path/to/skills ./anyclaw skill list
```

也可以在配置文件中设置：

```json
{
  "skills": {
    "dir": "skills",
    "auto_load": true,
    "include": [],
    "exclude": []
  }
}
```

## 权限字段

技能的 `permissions` 字段用于表达技能期望的权限范围。实际工具执行仍受 Agent 权限、工具策略、沙箱和安全配置共同约束。

常见权限：

| 权限 | 说明 |
| --- | --- |
| `read` | 读取文件或数据 |
| `write` | 写入文件或数据 |
| `exec` | 执行命令 |
| `network` | 访问网络 |

## 创建本地技能

```bash
mkdir -p skills/my-custom-skill
```

`skills/my-custom-skill/skill.json`：

```json
{
  "name": "my-custom-skill",
  "description": "Use this skill for custom project guidance.",
  "version": "1.0.0",
  "source": "local",
  "permissions": ["read"],
  "prompts": {
    "system": "Follow the project-specific guidance in this skill."
  }
}
```

`skills/my-custom-skill/SKILL.md`：

```markdown
# My Custom Skill

Use this skill when the user asks for project-specific guidance.
```

验证：

```bash
./anyclaw skill list
./anyclaw skill info my-custom-skill
```

## 故障排查

- 如果技能列表为空，确认 `skills.dir` 或 `ANYCLAW_SKILLS_DIR` 指向正确目录。
- 如果技能详情读取失败，检查 `skill.json` 是否为合法 JSON。
- 如果 Agent 没有按技能说明行动，确认核心说明是否写入了 `prompts.system` 或安装器是否生成了该字段。

## 下一步

- 阅读 [架构文档](ARCHITECTURE.md)
- 查看仓库内的 `skills/` 示例
