# HumanBehavior 反检测模块实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 通过模拟人类行为模式（随机延迟、曲线鼠标、真人打字），降低小红书账号被风控检测的风险。

**Architecture:** 新建 pkg/human/ 模块封装所有行为模拟方法，通过依赖注入方式在各处调用。核心算法包括贝塞尔曲线鼠标移动、随机间隔打字、元素内随机点击偏移。

**Tech Stack:** Go, go-rod, math/rand, time

---

## 文件结构

```
pkg/human/
├── human.go       # 核心模块：Config 结构体 + 5 个核心方法
└── human_test.go  # 单元测试

xiaohongshu/
├── publish.go        # 修改：约 20 处 Sleep + Click + Input + MoveTo
├── login.go          # 修改：约 5 处 Sleep
├── navigate.go       # 修改：约 2 处 MustClick → HumanClick
├── comment_feed.go   # 修改：约 3 处 Sleep
└── like_favorite.go  # 修改：约 3 处 Sleep
```

---

## Task 1: 创建 HumanBehavior 核心模块

**Files:**
- Create: `pkg/human/human.go`
- Test: `pkg/human/human_test.go`

- [ ] **Step 1: 创建 human.go 基础结构**

```go
package human

import (
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Config 反检测配置
type Config struct {
	DelayMin      time.Duration // 最小延迟
	DelayMax      time.Duration // 最大延迟
	TypingMin     time.Duration // 打字每个字符最小延迟
	TypingMax     time.Duration // 打字每个字符最大延迟
	ClickOffset   int           // 点击坐标随机偏移量（像素）
	EnableCurve   bool          // 启用曲线鼠标移动
	ScrollStepMin int           // 滚动步进最小值
	ScrollStepMax int           // 滚动步进最大值
}

// New 创建默认配置（low aggression）
func New() *Config {
	return &Config{
		DelayMin:      500 * time.Millisecond,
		DelayMax:      2000 * time.Millisecond,
		TypingMin:     50 * time.Millisecond,
		TypingMax:     150 * time.Millisecond,
		ClickOffset:   3,
		EnableCurve:   true,
		ScrollStepMin: 30,
		ScrollStepMax: 80,
	}
}

// SetAggression 设置 aggressiveness 等级
// - low:    延迟 500ms-2s, 打字 50-150ms, 偏移 ±3px
// - medium: 延迟 300ms-1.5s, 打字 30-100ms, 偏移 ±5px
// - high:   延迟 200ms-800ms, 打字 20-60ms, 偏移 ±8px
func (c *Config) SetAggression(level string) {
	switch level {
	case "medium":
		*c = Config{
			DelayMin:      300 * time.Millisecond,
			DelayMax:      1500 * time.Millisecond,
			TypingMin:     30 * time.Millisecond,
			TypingMax:     100 * time.Millisecond,
			ClickOffset:   5,
			EnableCurve:   true,
			ScrollStepMin: 20,
			ScrollStepMax: 60,
		}
	case "high":
		*c = Config{
			DelayMin:      200 * time.Millisecond,
			DelayMax:      800 * time.Millisecond,
			TypingMin:     20 * time.Millisecond,
			TypingMax:     60 * time.Millisecond,
			ClickOffset:   8,
			EnableCurve:   true,
			ScrollStepMin: 15,
			ScrollStepMax: 40,
		}
	default: // low
		*c = *New()
	}
}
```

- [ ] **Step 2: 添加 RandomDelay 方法**

```go
// RandomDelay 在配置范围内生成随机延迟并 sleep
func (c *Config) RandomDelay() {
	delay := c.DelayMin + time.Duration(rand.Intn(int(c.DelayMax-c.DelayMin)))
	time.Sleep(delay)
}

// Sleep 替代 time.Sleep，使用随机延迟
func (c *Config) Sleep() {
	c.RandomDelay()
}
```

- [ ] **Step 3: 添加贝塞尔曲线辅助函数**

```go
// cubicBezier 计算三次贝塞尔曲线上的点
func cubicBezier(t, p0, p1, p2, p3 float64) float64 {
	mt := 1 - t
	return mt*mt*mt*p0 + 3*mt*mt*t*p1 + 3*mt*t*t*p2 + t*t*t*p3
}

// randomFloat 生成范围内的随机浮点数
func randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
```

- [ ] **Step 4: 添加 HumanMoveTo 方法**

```go
// HumanMoveTo 使用曲线轨迹移动鼠标到目标位置
func (c *Config) HumanMoveTo(page *rod.Page, targetX, targetY float64) {
	if !c.EnableCurve {
		page.Mouse.MoveTo(targetX, targetY)
		return
	}

	pos := page.Mouse.Position()

	// 生成随机控制点（形成三次贝塞尔曲线）
	cp1x := pos.X + (targetX-pos.X)*0.3 + randomFloat(-50, 50)
	cp1y := pos.Y + randomFloat(-100, 100)
	cp2x := pos.X + (targetX-pos.X)*0.7 + randomFloat(-50, 50)
	cp2y := targetY + randomFloat(-100, 100)

	// 分段移动（10-20 步）
	steps := 10 + rand.Intn(10)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := cubicBezier(t, pos.X, cp1x, cp2x, targetX)
		y := cubicBezier(t, pos.Y, cp1y, cp2y, targetY)
		page.Mouse.MoveTo(x, y)
		time.Sleep(time.Duration(5+rand.Intn(10)) * time.Millisecond)
	}
}
```

- [ ] **Step 5: 添加 HumanClick 方法**

```go
// HumanClick 在元素中心+随机偏移位置点击
func (c *Config) HumanClick(elem *rod.Element) error {
	shape, err := elem.Shape()
	if err != nil {
		return elem.Click(proto.InputMouseButtonLeft, 1)
	}

	if len(shape.Quads) == 0 {
		return elem.Click(proto.InputMouseButtonLeft, 1)
	}

	quad := shape.Quads[0]
	minX, maxX := quad[0], quad[0]
	minY, maxY := quad[1], quad[1]
	for i := 0; i < quad.Len(); i++ {
		x := quad[i*2]
		y := quad[i*2+1]
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}

	// 中心点 + 随机偏移
	offsetX := float64(-c.ClickOffset+rand.Intn(2*c.ClickOffset)) + (maxX-minX)*0.3
	offsetY := float64(-c.ClickOffset+rand.Intn(2*c.ClickOffset)) + (maxY-minY)*0.3
	x := minX + (maxX-minX)/2 + offsetX
	y := minY + (maxY-minY)/2 + offsetY

	c.HumanMoveTo(elem.Page(), x, y)
	return elem.Page().Mouse.Click(proto.InputMouseButtonLeft, 1)
}
```

- [ ] **Step 6: 添加 HumanType 方法**

```go
// HumanType 模拟真人打字节奏
func (c *Config) HumanType(elem *rod.Element, text string) error {
	for _, ch := range text {
		if err := elem.Input(string(ch)); err != nil {
			return err
		}
		// 随机打字速度
		delay := c.TypingMin + time.Duration(rand.Intn(int(c.TypingMax-c.TypingMin)))
		time.Sleep(delay)

		// 5% 概率"思考"暂停
		if rand.Float32() < 0.05 {
			time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
		}
	}
	return nil
}
```

- [ ] **Step 7: 添加 HumanScroll 方法**

```go
// HumanScroll 分段滚动，模拟人类滚动行为
func (c *Config) HumanScroll(page *rod.Page, dy float64) {
	steps := 5 + rand.Intn(5)
	stepSize := dy / float64(steps)

	for i := 0; i < steps; i++ {
		scrollY := stepSize + randomFloat(-float64(c.ScrollStepMin), float64(c.ScrollStepMax))
		page.Mouse.Scroll(0, scrollY)
		time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)
	}
}
```

- [ ] **Step 8: 创建单元测试 human_test.go**

```go
package human

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	h := New()
	if h.DelayMin != 500*time.Millisecond {
		t.Errorf("expected DelayMin 500ms, got %v", h.DelayMin)
	}
	if h.DelayMax != 2000*time.Millisecond {
		t.Errorf("expected DelayMax 2000ms, got %v", h.DelayMax)
	}
	if h.ClickOffset != 3 {
		t.Errorf("expected ClickOffset 3, got %d", h.ClickOffset)
	}
}

func TestSetAggression(t *testing.T) {
	tests := []struct {
		level          string
		expectedMin    time.Duration
		expectedMax    time.Duration
		expectedOffset int
	}{
		{"low", 500 * time.Millisecond, 2000 * time.Millisecond, 3},
		{"medium", 300 * time.Millisecond, 1500 * time.Millisecond, 5},
		{"high", 200 * time.Millisecond, 800 * time.Millisecond, 8},
	}

	for _, tt := range tests {
		h := New()
		h.SetAggression(tt.level)
		if h.DelayMin != tt.expectedMin {
			t.Errorf("level %s: expected DelayMin %v, got %v", tt.level, tt.expectedMin, h.DelayMin)
		}
		if h.DelayMax != tt.expectedMax {
			t.Errorf("level %s: expected DelayMax %v, got %v", tt.level, tt.expectedMax, h.DelayMax)
		}
		if h.ClickOffset != tt.expectedOffset {
			t.Errorf("level %s: expected ClickOffset %d, got %d", tt.level, tt.expectedOffset, h.ClickOffset)
		}
	}
}

func TestCubicBezier(t *testing.T) {
	// t=0 时应该返回 p0
	result := cubicBezier(0, 0, 1, 2, 3)
	if result != 0 {
		t.Errorf("expected 0 at t=0, got %f", result)
	}

	// t=1 时应该返回 p3
	result = cubicBezier(1, 0, 1, 2, 3)
	if result != 3 {
		t.Errorf("expected 3 at t=1, got %f", result)
	}
}

func TestRandomFloat(t *testing.T) {
	min, max := 10.0, 20.0
	for i := 0; i < 100; i++ {
		r := randomFloat(min, max)
		if r < min || r > max {
			t.Errorf("randomFloat(%f, %f) = %f, out of range", min, max, r)
		}
	}
}
```

- [ ] **Step 9: 运行测试验证**

Run: `go test ./pkg/human/... -v`
Expected: PASS

- [ ] **Step 10: 提交**

```bash
git add pkg/human/human.go pkg/human/human_test.go
git commit -m "feat: add HumanBehavior module for anti-detection

- Add Config struct with delay, typing, click offset settings
- Implement RandomDelay, HumanMoveTo, HumanClick, HumanType, HumanScroll
- Add cubicBezier curve for human-like mouse movement
- Add SetAggression for low/medium/high profiles
- Add unit tests"
```

---

## Task 2: 修改 publish.go - 替换 Sleep 调用

**Files:**
- Modify: `xiaohongshu/publish.go:1-1293`

> **Note:** 需要在文件顶部添加 `human` 包导入，并在 submitPublish 函数开始处创建 human.Config 实例。

- [ ] **Step 1: 添加 human 包导入**

在 import 块添加：
```go
"github.com/xpzouying/xiaohongshu-mcp/pkg/human"
```

- [ ] **Step 2: 在 submitPublish 函数开头创建 human 实例**

在 `func submitPublish(...)` 函数开头（titleElem, err := page.Element 之前）添加：

```go
	h := human.New()
	// 可通过环境变量读取等级，暂时使用 low
	if level := os.Getenv("XHS_HUMAN_LEVEL"); level != "" {
		h.SetAggression(level)
	}
```

注意：需要添加 `os` 包导入。

- [ ] **Step 3: 替换固定 Sleep 调用**

搜索所有 `time.Sleep(` 并替换为 `h.Sleep()`，共约 20 处。

替换示例：
```go
// 原来
time.Sleep(1 * time.Second)
// 改为
h.Sleep()
```

注意：保留 `xiaohongshu/publish.go` 中 `waitForUploadComplete` 和 `waitForPublishButtonClickable` 函数里的轮询间隔（用于状态检测的快速轮询不应被人类行为干扰）。

- [ ] **Step 4: 验证编译**

Run: `go build ./...`
Expected: PASS (无编译错误)

- [ ] **Step 5: 提交**

```bash
git add xiaohongshu/publish.go
git commit -m "refactor(publish): replace fixed Sleep with human.RandomDelay"
```

---

## Task 3: 修改 publish.go - 替换打字和点击调用

**Files:**
- Modify: `xiaohongshu/publish.go`

- [ ] **Step 1: 替换 titleElem.Input 调用**

找到：
```go
titleElem.Input(title)
```

改为：
```go
h.HumanType(titleElem, title)
```

- [ ] **Step 2: 替换 contentElem.Input 调用**

找到：
```go
contentElem.Input(content)
```

改为：
```go
h.HumanType(contentElem, content)
```

- [ ] **Step 3: 替换 tag 输入打字调用**

在 `inputTag` 函数中，在函数开头获取 h 实例，并将 50ms 改为随机延迟。

- [ ] **Step 4: 验证编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add xiaohongshu/publish.go
git commit -m "refactor(publish): replace Input with HumanType for human-like typing"
```

---

## Task 4: 修改 publish.go - 替换鼠标移动和点击

**Files:**
- Modify: `xiaohongshu/publish.go`

- [ ] **Step 1: 在 clickPublishWidget 函数中使用 HumanMoveTo**

找到 `clickPublishWidget` 函数，将 `page.Mouse.MoveTo()` 改为 `h.HumanMoveTo()`。

- [ ] **Step 2: 验证编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add xiaohongshu/publish.go
git commit -m "refactor(publish): replace Mouse.MoveTo with HumanMoveTo"
```

---

## Task 5: 修改 login.go

**Files:**
- Modify: `xiaohongshu/login.go:1-101`

- [ ] **Step 1: 添加 human 包导入**

在 import 块添加：
```go
"github.com/xpzouying/xiaohongshu-mcp/pkg/human"
```

- [ ] **Step 2-4: 替换 CheckLoginStatus、Login、FetchQrcodeImage 中的 Sleep**

在每个函数开头创建 `h := human.New()` 实例，替换固定 Sleep 调用。

- [ ] **Step 5: 验证编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add xiaohongshu/login.go
git commit -m "refactor(login): replace fixed Sleep with human.RandomDelay"
```

---

## Task 6: 修改 navigate.go

**Files:**
- Modify: `xiaohongshu/navigate.go:1-45`

- [ ] **Step 1: 添加 human 包导入**

- [ ] **Step 2: 在 ToProfilePage 中替换 MustClick 为 HumanClick**

- [ ] **Step 3: 验证编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add xiaohongshu/navigate.go
git commit -m "refactor(navigate): replace MustClick with HumanClick"
```

---

## Task 7: 修改 comment_feed.go

**Files:**
- Modify: `xiaohongshu/comment_feed.go:1-273`

- [ ] **Step 1: 添加 human 包导入**

- [ ] **Step 2: 替换文件中的 time.Sleep 调用**

- [ ] **Step 3: 验证编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add xiaohongshu/comment_feed.go
git commit -m "refactor(comment): replace fixed Sleep with human.RandomDelay"
```

---

## Task 8: 修改 like_favorite.go

**Files:**
- Modify: `xiaohongshu/like_favorite.go:1-248`

- [ ] **Step 1: 添加 human 包导入**

- [ ] **Step 2: 替换文件中的 time.Sleep 调用**

- [ ] **Step 3: 验证编译**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add xiaohongshu/like_favorite.go
git commit -m "refactor(like): replace fixed Sleep with human.RandomDelay"
```

---

## Task 9: 端到端测试验证

- [ ] **Step 1: 运行所有测试**

Run: `go test ./... -v`
Expected: PASS

- [ ] **Step 2: 编译检查**

Run: `go build -o /dev/null .` (Linux/Mac) 或 `go build -o nul .` (Windows)
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add -A
git commit -m "test: run full test suite after human behavior refactor"
```

---

## Task 10: 更新 stealth 版本（可选）

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 检查最新 stealth 版本**

Run: `go list -m -versions github.com/go-rod/stealth`

- [ ] **Step 2: 更新到最新版本**

Run: `go get github.com/go-rod/stealth@latest`

- [ ] **Step 3: 验证**

Run: `go mod tidy && go build ./...`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add go.mod go.sum
git commit -m "chore: update go-rod/stealth to latest version"
```

---

## 验收检查清单

- [ ] HumanBehavior 模块独立可用
- [ ] 所有 `time.Sleep()` 被 `h.Sleep()` 替代（约 30+ 处）
- [ ] `Mouse.MoveTo()` 被 `HumanMoveTo()` 替代（约 1-2 处）
- [ ] `elem.Input()` 被 `HumanType()` 替代（约 3 处）
- [ ] `elem.Click()` 和 `MustClick()` 被 `HumanClick()` 替代（约 2-3 处）
- [ ] `go test ./...` 通过
- [ ] `go build ./...` 通过
- [ ] 所有改动已提交

---

## 执行方式选择

**Plan complete and saved to `docs/superpowers/plans/2026-06-01-human-behavior-implementation.md`**

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch 独立的 subagent 来执行每个任务，在任务之间 review，快速迭代

**2. Inline Execution** - 在本 session 中执行任务，使用 executing-plans 批量执行带检查点

**Which approach?**
