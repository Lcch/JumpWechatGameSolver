package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"sort"
	"time"
)

const (
	DEFAULT_T float64 = 738.0
	DEFAULT_X float64 = 450.0
	YMAX      int     = 1080
	ADB_PATH  string  = "/Users/lcch/Library/Android/sdk/platform-tools/adb"
)

type Point struct {
	x int
	y int
}

type ColorGroup struct {
	cnt int
	x   float64
	y   float64
}

func (c *ColorGroup) toString() string {
	return fmt.Sprintf("[%.2f, %.2f]", c.x, c.y)
}

type ImageSet interface {
	Set(x, y int, c color.Color)
}

func runBashCommand(cmd string) {
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Println(err)
	}
}

func screenShot() {
	runBashCommand(fmt.Sprintf("%s shell screencap /sdcard/screen.png", ADB_PATH))
	runBashCommand(fmt.Sprintf("%s pull /sdcard/screen.png .", ADB_PATH))
}

func readPng(filename string) [][][]uint8 {
	infile, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer infile.Close()

	img, _, err := image.Decode(infile)
	if err != nil {
		log.Fatal(err)
	}

	ret := make([][][]uint8, 0)

	b := img.Bounds()
	fmt.Println(b.Max.Y, b.Max.X)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := make([][]uint8, 0)
		for x := b.Min.X; x < b.Max.X; x++ {
			oldPixel := img.At(x, y)
			r, g, b, _ := oldPixel.RGBA()
			pixel := make([]uint8, 3, 3)
			pixel[0] = uint8(r)
			pixel[1] = uint8(g)
			pixel[2] = uint8(b)
			row = append(row, pixel)
		}
		ret = append(ret, row)
	}
	return ret
}

func getPivotPoints(img [][][]uint8) []*Point {
	pivots := make([]*Point, 0)
	w, h := len(img), len(img[0])
	for x := 500; x < w; x++ {
		for y := 0; y < h; y++ {
			r := img[x][y][0]
			g := img[x][y][1]
			b := img[x][y][2]
			if r == 58 && g == 58 && b == 102 {
				pivots = append(pivots, &Point{x: x, y: y})
			} else if r == 54 && g == 60 && b == 102 {
				pivots = append(pivots, &Point{x: x, y: y})
			} else if r == 57 && g == 57 && b == 99 {
				pivots = append(pivots, &Point{x: x, y: y})
			} else if r == 56 && g == 57 && b == 98 {
				pivots = append(pivots, &Point{x: x, y: y})
			}
		}
	}
	return pivots
}

func manhattanDistance(p, q *Point) float64 {
	return math.Abs(float64(p.x-q.x)) + math.Abs(float64(p.y-q.y))
}

func unionFind(fa []int, u int) int {
	if fa[u] == u {
		return u
	}
	fa[u] = unionFind(fa, fa[u])
	return fa[u]
}

func colorGroup(ps []*Point) []*ColorGroup {
	if len(ps) == 0 {
		return make([]*ColorGroup, 0)
	}
	num := len(ps)
	fa := make([]int, num, num)
	for i := 0; i < num; i++ {
		fa[i] = i
	}
	for i := 0; i < num; i++ {
		for j := 0; j < num; j++ {
			gi := unionFind(fa, i)
			gj := unionFind(fa, j)
			if gi != gj && manhattanDistance(ps[i], ps[j]) < 5 {
				fa[gj] = gi
			}
		}
	}
	groups := make([]*ColorGroup, 0)
	for i := 0; i < num; i++ {
		fa[i] = unionFind(fa, i)
	}
	for i := 0; i < num; i++ {
		if fa[i] != i {
			continue
		}
		newGroup := &ColorGroup{
			cnt: 0, x: 0, y: 0,
		}
		for j := 0; j < num; j++ {
			if fa[j] == i {
				newGroup.cnt += 1
				newGroup.x += float64(ps[j].x)
				newGroup.y += float64(ps[j].y)
			}
		}
		newGroup.x /= float64(newGroup.cnt)
		newGroup.y /= float64(newGroup.cnt)
		groups = append(groups, newGroup)
	}
	return groups
}

func majority(ps []*ColorGroup) []*ColorGroup {
	if len(ps) == 0 {
		return ps
	}
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].cnt > ps[j].cnt
	})
	return ps[:1]
}

func findPivotPoint(pivot []*Point) *ColorGroup {
	mp := majority(colorGroup(pivot))
	fmt.Printf("pivot %d: ", len(mp))
	for _, p := range mp {
		fmt.Printf("%d [%.2f, %.2f]; ", p.cnt, p.x, p.y)
	}
	fmt.Println()
	// Check pivot color cnt
	if math.Abs(float64(mp[0].cnt-370)) > 50 {
		log.Fatal("pivot num err")
	}
	return mp[0]
}

func makeJumpByT(t float64) {
	touch_command := fmt.Sprintf("%s shell input touchscreen swipe 250 323 250 323 %d", ADB_PATH, int(t))
	fmt.Println(touch_command)
	runBashCommand(touch_command)
}

func getTopPointOfNextBlock(img [][][]uint8, pivotPoint *ColorGroup) *ColorGroup {
	w, h := len(img), len(img[0])
	yleft := 0
	yright := h
	// 40 is to avoid getting pivot top as parts of next block.
	if pivotPoint.y > float64(YMAX/2) {
		yright = h/2 - 40
	} else {
		yleft = h/2 + 40
	}
	ret := &ColorGroup{
		cnt: 0, x: 0, y: 0,
	}
	for x := 500; x < w; x++ {
		ok := true
		for y := yleft; y < yright; y++ {
			disP := math.Abs(float64(img[x][y][0])-float64(img[x][0][0])) +
				math.Abs(float64(img[x][y][1])-float64(img[x][0][1])) +
				math.Abs(float64(img[x][y][2])-float64(img[x][0][2]))
			if disP > 10.0 {
				ok = false
				ret.cnt += 1
				ret.x += float64(x)
				ret.y += float64(y)
			}
		}
		if !ok {
			break
		}
	}
	ret.x /= math.Max(float64(ret.cnt), 1.0)
	ret.y /= math.Max(float64(ret.cnt), 1.0)
	return ret
}

func oneMove(step int) {
	img := readPng("screen.png")
	pivotBlocks := getPivotPoints(img)
	pivotPoint := findPivotPoint(pivotBlocks)
	fmt.Printf("pivot %s\n", pivotPoint.toString())
	topPoint := getTopPointOfNextBlock(img, pivotPoint)
	fmt.Printf("topPoint %s\n", topPoint.toString())
	makeJumpByT(math.Abs(pivotPoint.y-topPoint.y) / DEFAULT_X * DEFAULT_T + 4200.0 / math.Abs(topPoint.x - pivotPoint.x))
	// screen shot after make a move
	time.Sleep(1500 * time.Millisecond)
	screenShot()
	runBashCommand(fmt.Sprintf("cp screen.png %d.png", step))
}

func run() {
	screenShot()
	for i := 1; i < 1000; i++ {
		t := time.Now()
		fmt.Printf("step=%d\n", i)
		oneMove(i)
		fmt.Println(time.Since(t))
		fmt.Println()
	}
}

func main() {
	fs := flag.NewFlagSet("jump", flag.ExitOnError)
	method := fs.String("m", "new", "method: new|cont")
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	switch *method {
	case "new":
		run()
	case "cont":
		run()
	}
}
