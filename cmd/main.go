package main

import (
	"fmt"
	"github.com/goroom/free-proxy"
	"github.com/urfave/cli"
	"os"
)

func main() {

	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "proxy, p",
			Usage: "-p http://example:port",
		},
		cli.StringFlag{
			Name:  "bind, b",
			Usage: "-b :8080",
			Value: ":8080",
		},
		cli.StringFlag{
			Name:  "redis, r",
			Usage: "-r localhost:6379 (push to redis queue)",
		},
		cli.StringFlag{
			Name:  "rule, R",
			Usage: "-R rule.yml (filter rules)",
		},
		cli.Int64Flag{
			Name:  "log, l",
			Usage: "-l 1/2/3",
			Value: 1,
		},
		cli.StringFlag{
			Name:  "filter, f",
			Usage: "-f '*.js*'",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "sign",
			Aliases: []string{"s"},
			Usage:   "main sign example.com",
			Action: func(c *cli.Context) error {
				cproxy.GetCAPairPath(c.Args().First())
				return nil
			},
		},
	}
	app.Action = func(ctx *cli.Context) error {
		proxy := cproxy.NewProxy(
			ctx.String("bind"),
			ctx.String("redis"),
			ctx.String("rule"),
			ctx.String("proxy"),
			ctx.String("filter"),
		)
		proxy.Level = ctx.Int("log")
		if err:= proxy.Run();err!=nil{
			fmt.Println(err.Error())
			return err
		}
		return nil
	}
	app.Run(os.Args)
}
