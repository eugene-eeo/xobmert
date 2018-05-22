package main

import (
	"bufio"
	"flag"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/nsf/termbox-go"
)

func must(e error) {
	if e != nil {
		panic(e)
	}
}

type char struct {
	r       rune
	x, y    int
	fg, bg  termbox.Attribute
	t0      int
	temp    float64
	growth  float64
	burning bool
}

func (c *char) Style() (termbox.Attribute, termbox.Attribute) {
	return c.fg, c.bg
}

func (c *char) Rune() rune {
	return c.r
}

func (c *char) Update(t int) bool {
	if !c.burning {
		return true
	}
	// approximate some kind of exponential temperature growth,
	// also update rune so that it looks like the charcater is burning
	c.temp += c.growth * float64((t-c.t0)/2+1)
	c.r += rune(rand.Intn(10))
	if c.temp < 5 {
		c.bg = termbox.ColorYellow
	} else if c.temp < 10 {
		c.bg = termbox.ColorRed
	} else if c.temp < 15 {
		c.bg = termbox.ColorBlue
	} else {
		return false
	}
	return true
}

func (c *char) burn(t int) {
	c.t0 = t
	c.burning = true
	c.growth = rand.Float64() * CONFIG.growth_scaling
	c.temp = rand.Float64() * CONFIG.temp_scaling
}

func draw(grid [][]*char) {
	for y, row := range grid {
		for x, c := range row {
			if c != nil {
				f, b := c.Style()
				termbox.SetCell(x, y, c.Rune(), f, b)
			}
		}
	}
}

func flameProb(grid [][]*char, x0, y0 int) float64 {
	// all particles have a resting probability of being
	// spontaneously lit alight. Then probability of being
	// lit alight will be depend on if it's surrounding
	// cells are on fire.
	f := CONFIG.spontaneous
	k := 0.0
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
			if c != nil && c.burning {
				k += c.temp
			}
		}
	}
	return f * math.Pow(CONFIG.adjacent_factor, float64(k))
}

func loop(grid [][]*char, events chan termbox.Event) {
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
					if !c.burning && rand.Float64() <= flameProb(grid, x, y) {
						c.burn(t)
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

type config struct {
	spontaneous     float64
	adjacent_factor float64
	temp_scaling    float64
	growth_scaling  float64
}

var CONFIG *config = &config{
	spontaneous:     0.04,
	adjacent_factor: 1.54,
	temp_scaling:    2.0,
	growth_scaling:  1.0,
}

func main() {
	file := flag.String("f", "", "file to read from")
	flag.Float64Var(&CONFIG.spontaneous, "sp", CONFIG.spontaneous, "spontaneous combustion probability")
	flag.Float64Var(&CONFIG.adjacent_factor, "adj", CONFIG.adjacent_factor, "effect of adjacent flames")
	flag.Float64Var(&CONFIG.temp_scaling, "ts", CONFIG.temp_scaling, "temperature scaling")
	flag.Float64Var(&CONFIG.growth_scaling, "gs", CONFIG.growth_scaling, "temp growth scaling")
	flag.Parse()

	f, err := os.Open(*file)
	must(err)

	rand.Seed(int64(time.Now().Nanosecond()))
	must(termbox.Init())
	defer termbox.Close()

	w, h := termbox.Size()
	grid := make([][]*char, h)
	for i := 0; i < h; i++ {
		grid[i] = make([]*char, w)
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
