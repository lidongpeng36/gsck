package main

import (
	"fmt"

	"github.com/lidongpeng36/fc/command"
	"github.com/codegangsta/cli"
)

func init() {
	app := command.Instance()
	app.Name = "test"
	app.Author = "lidongpeng36@gmail.com"
	app.Version = "0.0.1"
	app.Usage = "Test"
	app.Action = action
}

func action(c *cli.Context) {
	app := command.Instance()
	fmt.Println("PIPE: ", app.Pipe)
	fmt.Println("FIFO: ", app.Fifo)
	fmt.Println("FIFOFile: ", app.FifoFile)
}

func main() {
	command.Run()
}
