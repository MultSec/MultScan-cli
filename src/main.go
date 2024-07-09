package main

import (
    "fmt"
    "os"

    "github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name: "MultScan CLI client",

        Commands: []*cli.Command{
            {
                Name:  "machines",
                Usage: "Retrieve list of machines present",

                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:        "server",
                        Aliases:     []string{"s"},
                        Value:       "127.0.0.1",
                        Usage:       "Use provided `IP` for the MultScan server",
                        DefaultText: "127.0.0.1",
                    },
                    &cli.IntFlag{
                        Name:        "port",
                        Aliases:     []string{"p"},
                        Value:       8000,
                        Usage:       "Use provided `PORT` for the MultScan server",
                        DefaultText: "8000",
                    },
                },
                Action: func(ctx *cli.Context) error {
                    machines, err  := getMachines(ctx.String("server"), ctx.Int("port"))

                    if err != nil {
                        printLog(logError, fmt.Sprintf("%v", err))
                        return nil
                    }

                    displayMachines(machines)

                    return nil
                },
            },
            {
                Name:  "scan",
                Usage: "Scans the provided executable across the available vms",

                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:        "server",
                        Aliases:     []string{"s"},
                        Value:       "127.0.0.1",
                        Usage:       "Use provided `IP` for the MultScan server",
                        DefaultText: "127.0.0.1",
                    },
                    &cli.IntFlag{
                        Name:        "port",
                        Aliases:     []string{"p"},
                        Value:       8000,
                        Usage:       "Use provided `PORT` for the MultScan server",
                        DefaultText: "8000",
                    },
                    &cli.StringFlag{
                        Name:        "exe",
                        Aliases:     []string{"e"},
                        Usage:       "Executable to be scanned `FILE`",
                        Required:    true,
                    },
                },
                Action: func(ctx *cli.Context) error {
                    _, err  := getMachines(ctx.String("server"), ctx.Int("port"))

                    if err != nil {
                        printLog(logError, fmt.Sprintf("%v", err))
                        return nil
                    }

                    return nil
                },
            },
        },
    }

    if err := app.Run(os.Args); err != nil {
        printLog(logError, fmt.Sprintf("%v", err))
    }
}