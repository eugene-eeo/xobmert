package main

import "math"
import "time"
import "math/rand"
import "bufio"
import "os"
import "github.com/nsf/termbox-go"

func must(e error) {
	if e != nil {
		panic(e)
	}
}

type renderable interface {
	Style() (termbox.Attribute, termbox.Attribute)
	Rune() rune
	Update(t int) bool
	IsSpark() bool
}

type char struct {
	r      rune
	x, y   int
	fg, bg termbox.Attribute
}

func (c *char) IsSpark() bool { return false }

func (c *char) Update(i int) bool { return true }

func (c *char) Style() (termbox.Attribute, termbox.Attribute) {
	return c.fg, c.bg
}

func (c *char) Rune() rune {
	return c.r
}

type spark struct {
	*char
	time   int
	temp   float64
	growth float64
}

func (s *spark) IsSpark() bool { return true }

func (s *spark) Update(t int) bool {
	// initial update; do nothing on the first tick.
	if s.time == 0 {
		s.time = t
		return true
	}
	// approximate some kind of exponential temperature growth,
	// also update rune so that it looks like the charcater is burning
	s.temp += s.growth * float64((t-s.time)/2+1)
	s.r += rune(rand.Intn(10))
	if s.temp < 5 {
		s.bg = termbox.ColorYellow
	} else if s.temp < 10 {
		s.bg = termbox.ColorRed
	} else if s.temp < 15 {
		s.bg = termbox.ColorBlue
	} else {
		return false
	}
	return true
}

func draw(grid [][]renderable) {
	for y, row := range grid {
		for x, c := range row {
			if c != nil {
				f, b := c.Style()
				termbox.SetCell(x, y, c.Rune(), f, b)
			}
		}
	}
}

func newSparkFromChar(c *char) *spark {
	return &spark{
		char:   c,
		growth: rand.Float64(),
		temp:   0,
	}
}

func flameProb(grid [][]renderable, x0, y0 int) float64 {
	// all particles have a resting probability of being
	// spontaneously lit alight. Then probability of being
	// lit alight will be depend on if it's surrounding
	// cells are on fire.
	f := 0.04
	k := 0
	D := []int{-1, 0, 1}
	for _, dx := range D {
		x := x0 + dx
		if x < 0 || x == len(grid[0]) {
			continue
		}
		for _, dy := range D {
			y := y0 + dy
			if y < 0 || y == len(grid) {
				continue
			}
			c := grid[y][x]
			if c != nil && c.IsSpark() {
				k += 1
			}
		}
	}
	return f * math.Pow(1.54, float64(k))
}

func loop(grid [][]renderable, events chan termbox.Event) {
	ticks := time.NewTicker(time.Millisecond * 100)
	t := 0
	for {
		select {
		case <-events:
			return
		case <-ticks.C:
			t++
			stop := true
			for y, row := range grid {
				for x, c := range row {
					if c == nil {
						continue
					}
					stop = false
					if !c.IsSpark() && rand.Float64() <= flameProb(grid, x, y) {
						grid[y][x] = newSparkFromChar(c.(*char))
					}
					if !c.Update(t) {
						grid[y][x] = nil
					}
				}
			}
			// automatically exit when there are no particles left
			// so we don't aimlessly spin on CPU
			if stop {
				return
			}
			must(termbox.Clear(termbox.ColorDefault, termbox.ColorDefault))
			draw(grid)
			must(termbox.Flush())
		}
	}
}

func main() {
	files := os.Args[1:]
	if len(files) == 0 {
		os.Exit(1)
	}
	f, err := os.Open(files[0])
	if err != nil {
		panic(err)
	}
	rand.Seed(int64(time.Now().Nanosecond()))
	must(termbox.Init())
	defer termbox.Close()

	w, h := termbox.Size()
	grid := make([][]renderable, h)
	for i := 0; i < h; i++ {
		grid[i] = make([]renderable, w)
	}

	r := bufio.NewScanner(f)
	for y := 0; r.Scan() && y < h; y++ {
		line := r.Text()
		dx := 0
		for x, c := range line {
			if x+dx >= w {
				break
			}
			switch c {
			case '\t':
				dx += 3
			case ' ', '\n':
				break
			default:
				grid[y][x+dx] = &char{
					r:  c,
					fg: termbox.ColorDefault,
					bg: termbox.ColorDefault,
				}
			}
		}
	}

	events := make(chan termbox.Event)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()
	loop(grid, events)
}
