package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	robot "github.com/go-vgo/robotgo"
)

type Point struct {
	x int
	y int
}

var (
	_urls []string
)


var (
	counter = 1
	LinkPoint = Point{0,0}
	ClosePoint = Point{0,0}
	InputPoint = Point{0,0}

	stop				bool
	sleepSecond 		int
)

func initFlag()  {
	flag.IntVar(&sleepSecond, "s", 5, "sleep second")
	flag.Parse()
}

func initPoint()  {
	fmt.Println("鼠标移动到点击链接的位置(f3结束)")
	key := robot.AddEvent("f3")
	sleep(1)
	if key == 0 {
		x, y := robot.GetMousePos()
		LinkPoint = Point{x, y}
		fmt.Println("文章链接坐标", x, y)
	}
	fmt.Println("鼠标移动到输入框的位置(f3结束)")
	key = robot.AddEvent("f3")
	sleep(1)
	if key == 0 {
		x, y := robot.GetMousePos()
		InputPoint = Point{x, y}
		fmt.Println("输入框坐标", x, y)
	}
	fmt.Println("程序开始,点击F8退出")
}

func initAccounts()  {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	file := filepath.Join(dir, "accounts.txt")
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	br := bufio.NewReader(f)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		_urls = append(_urls, string(a))
	}
}

func stopRobot()  {
	for {
		key := robot.AddEvent("f8")
		sleep(1)
		if key == 0 {
			stop = !stop
			if stop {
				log.Println("程序停止")
			} else {
				log.Println("程序开始")
			}
		}
	}
}

func process()  {
	fmt.Printf("已经访问个数: %d %s \n", counter, time.Now().String())
	nextUrl := NextUrl()
	if nextUrl == "" {
		return
	}
	robot.TypeString(nextUrl)
	robot.KeyTap("enter")
	robot.MoveMouseSmooth(LinkPoint.x, LinkPoint.y, 0.5)
	robot.Click("left", false)

	robot.MoveMouseSmooth(InputPoint.x, InputPoint.y, 0.5)
	robot.Click("left", false)

	counter = counter + 1

}

func NextUrl() string {
	return _urls[counter -1]
}

func init()  {
	initFlag()
	initPoint()
	initAccounts()
}


func main()  {
	go stopRobot()
	for {
		if stop {
			time.Sleep(1 * time.Second)
			continue
		}
		process()
		sleep(sleepSecond)
	}
}


func sleep(n int)  {
	time.Sleep(time.Duration(n) * time.Second)
}
