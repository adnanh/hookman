package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/adnanh/hookman/parser"
	"github.com/adnanh/webhook/hook"
	"github.com/codegangsta/cli"
)

const (
	version = "0.0.1"
)

var authors = []cli.Author{
	{
		Name:  "Adnan Hajdarevic",
		Email: "adnanh@gmail.com",
	},
}

var webhookHooks hook.Hooks
var hooksFile string
var hookCount int

func loadHooksOrDie(c *cli.Context) {
	hooksFile = c.GlobalString("file")
	err := webhookHooks.LoadFromFile(hooksFile)

	if err != nil {
		log.Fatalf("error: could not load hooks from file: %+v\n", err)
	}

	hookCount = len(webhookHooks)
}

func listAllHooks(c *cli.Context) {
	loadHooksOrDie(c)

	if hookCount == 0 {
		log.Fatalln("hooks file is empty")
	}

	hooksMap := make(map[string]hook.Hooks)
	var hooksIds []string

	for _, h := range webhookHooks {
		if _, ok := hooksMap[h.ID]; !ok {
			hooksIds = append(hooksIds, h.ID)
		}
		hooksMap[h.ID] = append(hooksMap[h.ID], h)
	}

	sort.Strings(hooksIds)

	compact := c.BoolT("compact")

	for _, hookID := range hooksIds {
		for idx, h := range hooksMap[hookID] {
			if compact {
				log.Printf("%d, %s\n", idx, (CompactHook)(h))
			} else {
				log.Printf("INDEX:\n   %d\n\n%s\n", idx, (Hook)(h))
			}
		}
	}

	if compact {
		log.Println("")
	}

	log.Printf("total %d hook(s) in file: %s\n", hookCount, hooksFile)
}

func editHook(c *cli.Context) {
	loadHooksOrDie(c)

	p := parser.New(strings.Join(c.Args(), " "))

	if error := p.Parse(); error != nil {
		fmt.Printf("%+v\n", error)
		return
	}
}

func saveToHooksFile() {
	formattedOutput, err := json.MarshalIndent(webhookHooks, "", "  ")

	if err != nil {
		log.Fatalf("error: could not format hooks file:\n%+v\n", err)
	}

	fileInfo, _ := os.Stat(hooksFile)
	fileMode := fileInfo.Mode()

	err = ioutil.WriteFile(hooksFile, formattedOutput, fileMode.Perm())

	if err != nil {
		log.Fatalf("error: could not create hooks file:\n%+v\n", err)
	} else {
		log.Println("success")
	}
}

func formatHooksFile(c *cli.Context) {
	loadHooksOrDie(c)
	saveToHooksFile()
}

func showHook(c *cli.Context) {
	if len(c.Args()) == 0 {
		log.Fatalln("error: you must provide a valid hook id")
	}

	loadHooksOrDie(c)

	var hooksSlice hook.Hooks

	hookID := c.Args()[0]

	for _, h := range webhookHooks {
		if h.ID == hookID {
			hooksSlice = append(hooksSlice, h)
		}
	}

	matchingHookCount := len(hooksSlice)

	if matchingHookCount == 0 {
		log.Fatalln("error: could not find any hooks matching the given id")
	}

	if c.IsSet("idx") {
		idx := c.Int("idx")

		if idx >= matchingHookCount || idx < 0 {
			log.Fatalln("error: given local hook index is out of bounds")
		}

		log.Printf("INDEX:\n   %d\n\n%s\n", idx, (Hook)(hooksSlice[idx]))
	} else {
		for idx, hook := range hooksSlice {
			log.Printf("INDEX:\n   %d\n\n%s\n", idx, (Hook)(hook))
		}
	}
}

func init() {
	log.SetFlags(0)
}

/*
	 hookman add|a redeploy
		 adds an empty hook named redeploy

	 hookman remove|rm|delete|del redeploy
		 --all - removes all hooks
		 --idx 1 - remove the 2nd instance of redeploy hook
		 removes the hook named redeploy

	 hookman edit|e redeploy
		 --create create hook if it does not exist
		 --set property=value set value for the given property
		 --append property=value append the value to the property
		 --unset property[idx] removes the property from the hook, or the idx-th value from the property
		 modifies the hook named redeploy based on the flags
*/

func main() {
	app := cli.NewApp()

	app.Name = "hookman"
	app.Version = version
	app.Authors = authors
	app.Usage = "manage webhook hooks file"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "file, f",
			Value:  "hooks.json",
			Usage:  "path to the hooks file",
			EnvVar: "HOOKS_FILE",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "prints all hooks in the hooks file",
			Action:  listAllHooks,
			Flags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "compact, c",
					Usage: "print compact version of hooks",
				},
			},
		},
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "shows the given hook details",
			Action:  showHook,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "idx, i",
					Usage: "local hook index (used for differentiating multiple hooks with the same id)",
				},
			},
		},
		{
			Name:    "format",
			Aliases: []string{"fmt"},
			Usage:   "cleans up and reindents hooks file",
			Action:  formatHooksFile,
		},
		{
			Name:    "edit",
			Aliases: []string{"e"},
			Usage:   "modifies the given hook according to specified flags",
			Action:  editHook,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "idx, i",
					Value: 0,
					Usage: "local hook index (used for differentiating multiple hooks with the same id)",
				},
				cli.StringSliceFlag{
					Name:  "set, s",
					Usage: "property=value",
				},
				cli.StringSliceFlag{
					Name:  "append, a",
					Usage: "property=value",
				},
				cli.StringSliceFlag{
					Name:  "unset, u",
					Usage: "property name",
				},
			},
		},
	}

	app.Run(os.Args)
}
