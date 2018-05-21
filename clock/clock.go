package main

import "time"
import "github.com/nsf/termbox-go"

type grid [5][3]bool

var SEP grid = grid([5][3]bool{
	{false, false, false},
	{false, true, false},
	{false, false, false},
	{false, true, false},
	{false, false, false},
})

func (g *grid) draw(dx int, dy int) {
	for x := 0; x < 3; x++ {
		for y := 0; y < 5; y++ {
			if g[y][x] {
				termbox.SetCell(x+dx, y+dy, ' ', termbox.ColorDefault, termbox.ColorWhite)
			}
		}
	}
}

func intToGrid(n int) grid {
	return [5][3]bool{
		{true, n != 4, n != 1},
		{n == 0 || (n > 3 && n != 7), n == 1, n != 1 && n != 5},
		{n != 1 && n != 7, n != 0 && n != 7, n != 1},
		{n == 0 || n == 2 || n == 6 || n >= 8, n == 1, n != 1 && n != 2},
		{n != 4 && n != 7, n != 4 && n != 7, true},
	}
}

type clock struct {
	width  int
	height int
	now    time.Time
}

func (c *clock) tick() {
	_ = termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	hour := c.now.Hour()
	mins := c.now.Minute()
	h1, h2 := intToGrid(hour/10), intToGrid(hour%10)
	m1, m2 := intToGrid(mins/10), intToGrid(mins%10)

	dy := c.height/2 - 2
	dx := c.width/2 - 8

	h1.draw(dx+0, dy)
	h2.draw(dx+4, dy)
	SEP.draw(dx+7, dy)
	m1.draw(dx+10, dy)
	m2.draw(dx+14, dy)

	_ = termbox.Flush()
}

func (c *clock) loop(events chan termbox.Event) {
	t := time.NewTicker(time.Second * 30)
	for {
		c.tick()
		select {
		case e := <-events:
			switch e.Type {
			case termbox.EventKey:
				t.Stop()
				return
			case termbox.EventResize:
				c.width = e.Width
				c.height = e.Height
			}
		case now := <-t.C:
			c.now = now
		}
	}
}

func main() {
	_ = termbox.Init()
	defer termbox.Close()
	width, height := termbox.Size()
	c := &clock{
		width:  width,
		height: height,
		now:    time.Now(),
	}
	events := make(chan termbox.Event)
	go func() {
		for {
			evt := termbox.PollEvent()
			events <- evt
		}
	}()
	c.loop(events)
}
