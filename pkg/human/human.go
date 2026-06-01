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

// RandomDelay 在配置范围内生成随机延迟并 sleep
func (c *Config) RandomDelay() {
	delay := c.DelayMin + time.Duration(rand.Intn(int(c.DelayMax-c.DelayMin)))
	time.Sleep(delay)
}

// Sleep 替代 time.Sleep，使用随机延迟
func (c *Config) Sleep() {
	c.RandomDelay()
}

// cubicBezier 计算三次贝塞尔曲线上的点
func cubicBezier(t, p0, p1, p2, p3 float64) float64 {
	mt := 1 - t
	return mt*mt*mt*p0 + 3*mt*mt*t*p1 + 3*mt*t*t*p2 + t*t*t*p3
}

// randomFloat 生成范围内的随机浮点数
func randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// HumanMoveTo 使用曲线轨迹移动鼠标到目标位置
func (c *Config) HumanMoveTo(page *rod.Page, targetX, targetY float64) {
	if !c.EnableCurve {
		page.Mouse.MoveTo(proto.Point{X: targetX, Y: targetY})
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
		page.Mouse.MoveTo(proto.Point{X: x, Y: y})
		time.Sleep(time.Duration(5+rand.Intn(10)) * time.Millisecond)
	}
}

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

// HumanScroll 分段滚动，模拟人类滚动行为
func (c *Config) HumanScroll(page *rod.Page, dy float64) {
	steps := 5 + rand.Intn(5)
	stepSize := dy / float64(steps)

	for i := 0; i < steps; i++ {
		scrollY := stepSize + randomFloat(-float64(c.ScrollStepMin), float64(c.ScrollStepMax))
		page.Mouse.Scroll(0, scrollY, 1)
		time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)
	}
}
