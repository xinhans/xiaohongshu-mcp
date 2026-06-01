# HumanBehavior 反检测模块设计

## 1. 概述

**目标**：通过模拟人类行为模式，降低小红书账号被风控检测的风险。

**核心问题**：当前代码使用大量固定时间间隔和直线鼠标移动，容易被识别为机器人操作。

**解决方案**：封装 HumanBehavior 模块，提供随机延迟、曲线移动、真人打字等行为模拟能力。

---

## 2. 技术方案

### 2.1 模块位置

```
pkg/human/
├── human.go      # 核心实现
└── human_test.go # 单元测试
```

### 2.2 核心接口

```go
package human

// Config 反检测配置
type Config struct {
    // DelayRange 操作间隔时间范围
    DelayMin time.Duration
    DelayMax time.Duration

    // TypingDelayPerChar 打字每个字符的延迟范围
    TypingMin time.Duration
    TypingMax time.Duration

    // ClickOffset 点击坐标随机偏移量（像素）
    ClickOffset int

    // EnableCurve 启用曲线鼠标移动
    EnableCurve bool

    // ScrollStep 滚动步进随机范围
    ScrollStepMin int
    ScrollStepMax int
}

// New 默认配置（low aggression）
func New() *Config

// SetAggression 设置 aggressiveness 等级
// - low:    延迟 500ms-2s, 打字 50-150ms, 偏移 ±3px
// - medium: 延迟 300ms-1.5s, 打字 30-100ms, 偏移 ±5px
// - high:   延迟 200ms-800ms, 打字 20-60ms, 偏移 ±8px
func (c *Config) SetAggression(level string)

// RandomDelay 在配置范围内生成随机延迟
func (c *Config) RandomDelay()

// HumanMoveTo 使用曲线轨迹移动鼠标到目标位置
func (c *Config) HumanMoveTo(page *rod.Page, x, y float64)

// HumanClick 在元素中心+随机偏移位置点击
func (c *Config) HumanClick(elem *rod.Element)

// HumanType 模拟真人打字节奏
func (c *Config) HumanType(elem *rod.Element, text string)

// HumanScroll 分段滚动，模拟人类滚动行为
func (c *Config) HumanScroll(page *rod.Page, dy float64)

// Sleep 替代 time.Sleep，使用随机延迟
func (c *Config) Sleep()
```

### 2.3 贝塞尔曲线移动算法

```go
func (c *Config) HumanMoveTo(page *rod.Page, targetX, targetY float64) {
    // 1. 获取当前位置
    pos := page.Mouse.Position()

    // 2. 生成随机控制点（形成三次贝塞尔曲线）
    cp1x := pos.X + (targetX-pos.X)*0.3 + randomFloat(-50, 50)
    cp1y := pos.Y + randomFloat(-100, 100)
    cp2x := pos.X + (targetX-pos.X)*0.7 + randomFloat(-50, 50)
    cp2y := targetY + randomFloat(-100, 100)

    // 3. 分段移动（10-20 步）
    steps := 10 + rand.Intn(10)
    for i := 1; i <= steps; i++ {
        t := float64(i) / float64(steps)
        x := bezierPoint(t, pos.X, cp1x, cp2x, targetX)
        y := bezierPoint(t, pos.Y, cp1y, cp2y, targetY)
        page.Mouse.MoveTo(x, y)
        time.Sleep(time.Duration(5+rand.Intn(10)) * time.Millisecond)
    }
}
```

### 2.4 真人打字算法

```go
func (c *Config) HumanType(elem *rod.Element, text string) {
    for _, ch := range text {
        elem.Input(string(ch))
        // 随机打字速度
        delay := c.TypingMin + time.Duration(rand.Intn(int(c.TypingMax-c.TypingMin)))
        time.Sleep(delay)

        // 5% 概率"思考"暂停
        if rand.Float32() < 0.05 {
            time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
        }
    }
}
```

---

## 3. 改动范围

### 3.1 新增文件

| 文件 | 说明 |
|------|------|
| `pkg/human/human.go` | 核心模块 |
| `pkg/human/human_test.go` | 单元测试 |

### 3.2 修改文件

| 文件 | 改动内容 |
|------|---------|
| `xiaohongshu/publish.go` | 替换 `time.Sleep()` → `human.Sleep()` |
|  | 替换 `elem.Click()` → `human.HumanClick()` |
|  | 替换 `elem.Input(text)` → `human.HumanType()` |
|  | 替换 `page.Mouse.MoveTo()` → `human.HumanMoveTo()` |
| `xiaohongshu/login.go` | 替换 `time.Sleep()` → `human.Sleep()` |
| `xiaohongshu/navigate.go` | 替换直线移动 → `human.HumanMoveTo()` |
| `xiaohongshu/comment_feed.go` | 替换 `time.Sleep()` → `human.Sleep()` |
| `xiaohongshu/like_favorite.go` | 替换 `time.Sleep()` → `human.Sleep()` |

### 3.3 配置化改造

在 `configs/` 下新增人类行为配置：

```go
// configs/human.go
package config

var HumanBehaviorConfig = &human.Config{
    DelayMin:   500 * time.Millisecond,
    DelayMax:   2000 * time.Millisecond,
    TypingMin:  50 * time.Millisecond,
    TypingMax:  150 * time.Millisecond,
    ClickOffset: 3,
    EnableCurve: true,
}
```

---

## 4. 使用示例

```go
// 在 service.go 或 handlers 中初始化
h := human.New()
h.SetAggression("low") // 或从环境变量读取

// 发布时使用
func submitPublish(page *rod.Page, ...) error {
    h.Sleep() // 替代 time.Sleep(1 * time.Second)

    titleElem, _ := page.Element("div.d-input input")
    h.HumanType(titleElem, title) // 替代 titleElem.Input(title)

    // ...
}
```

---

## 5. 测试策略

| 测试类型 | 内容 |
|---------|------|
| 单元测试 | 贝塞尔曲线计算、随机延迟范围、打字节奏验证 |
| 集成测试 | 实际浏览器操作，观察行为是否符合预期 |
| 回归测试 | 确保改动不破坏现有功能 |

---

## 6. 风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| 过度随机化导致不稳定 | 提供配置开关，允许关闭某些特性 |
| 性能下降 | 延迟上限设置合理（不超过 2s） |
| stealth 版本过时 | 定期更新 go-rod/stealth 依赖 |

---

## 7. 优先级

1. **P0**: HumanBehavior 核心模块开发
2. **P0**: publish.go 全面改造
3. **P1**: login.go 和 navigate.go 改造
4. **P1**: 其他文件的 Sleep 替换
5. **P2**: 配置化改造，支持环境变量
6. **P2**: 添加测试

---

## 8. 验收标准

- [ ] HumanBehavior 模块可独立使用
- [ ] 所有 `time.Sleep()` 被随机延迟替代
- [ ] 鼠标移动使用曲线轨迹
- [ ] 打字操作模拟真人节奏
- [ ] 可通过环境变量配置 aggressiveness 等级
- [ ] 现有功能测试通过
