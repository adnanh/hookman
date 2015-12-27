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

var (
	authors = []cli.Author{
		{
			Name:  "Adnan Hajdarevic",
			Email: "adnanh@gmail.com",
		},
	}

	hooks     hook.Hooks
	hooksIds  []string
	hooksMap  = make(map[string][]*hook.Hook)
	hooksFile string
)

func deleteHooks(hooksToDelete []*hook.Hook) {
	var newHooks hook.Hooks

	for i := 0; i < len(hooks); i++ {
		found := false

		for j := 0; j < len(hooksToDelete); j++ {
			if &(hooks[i]) == hooksToDelete[j] {
				found = true
			}
		}

		if !found {
			newHooks = append(newHooks, hooks[i])
		}
	}

	hooks = newHooks
}

func loadHooks(c *cli.Context) error {
	hooksFile = c.GlobalString("file")

	if err := hooks.LoadFromFile(hooksFile); err != nil {
		return fmt.Errorf("could not load hooks from file: %s\n", err)
	}

	for i := 0; i < len(hooks); i++ {
		h := &hooks[i]

		if _, ok := hooksMap[h.ID]; !ok {
			hooksIds = append(hooksIds, h.ID)
		}

		hooksMap[h.ID] = append(hooksMap[h.ID], h)
	}

	sort.Strings(hooksIds)

	return nil
}

func saveHooks() error {
	formattedOutput, err := json.MarshalIndent(hooks, "", "  ")

	if err != nil {
		return fmt.Errorf("could not format hooks file: %s\n", err)
	}

	fileInfo, _ := os.Stat(hooksFile)
	fileMode := fileInfo.Mode()

	err = ioutil.WriteFile(hooksFile, formattedOutput, fileMode.Perm())

	if err != nil {
		return fmt.Errorf("could not create hooks file: %s\n", err)
	}

	log.Println("ok")

	return nil
}

func formatHooksFile(c *cli.Context) {
	if err := loadHooks(c); err != nil {
		log.Fatalf("error: %s\n", err)
	}

	if err := saveHooks(); err != nil {
		log.Fatalf("error: %s\n", err)
	}
}

func printHook(h *hook.Hook, idx int, compact bool) {
	if compact {
		log.Printf("  %s\n\n", fmt.Sprintf((*CompactHook)(h).String(), idx))
	} else {
		log.Printf("INDEX:\n   %d\n\n%s\n", idx, (*Hook)(h))
	}
}

func terminateOnEmptyHooksFile() {
	if len(hooks) == 0 {
		log.Fatalln("error: hooks file is empty")
	}
}

func listHooks(c *cli.Context) {
	if err := loadHooks(c); err != nil {
		log.Fatalf("error: %s\n", err)
	}

	terminateOnEmptyHooksFile()

	expanded := c.Bool("expanded")

	if len(c.Args()) == 0 {
		// user did not supply hook id, print all hooks
		for _, hookID := range hooksIds {
			for idx, h := range hooksMap[hookID] {
				printHook(h, idx, !expanded)
			}
		}

		log.Printf("total %d hook(s) in file: %s\n", len(hooks), hooksFile)
	} else {
		// user supplied hook id, print only hooks matching the given id

		compact := c.Bool("compact")

		if c.IsSet("idx") {
			h, err := findOneHookByID(c)

			if err != nil {
				log.Fatalf("error: %s\n", err)
			}

			printHook(h, c.Int("idx"), compact)
		} else {
			hooksSlice, err := findHooksByID(c)

			if err != nil {
				log.Fatalf("error: %s\n", err)
			}

			for idx, h := range hooksSlice {
				printHook(h, idx, compact)
			}

			log.Printf("total %d hook(s) with ID %s in file: %s\n", len(hooksSlice), c.Args()[0], hooksFile)
		}

	}
}

func addHook(c *cli.Context) {
	if err := loadHooks(c); err != nil {
		log.Fatalf("error: %s\n", err)
	}

	if len(c.Args()) == 0 {
		log.Fatalln("error: you must supply hook id")
	}

	newHook := hook.Hook{ID: c.Args()[0]}

	if err := setHookProperties(&newHook, c.StringSlice("set")); err != nil {
		log.Fatalf("error: cannot set property: %s\n", err)
	}

	hooks = append(hooks, newHook)

	if err := saveHooks(); err != nil {
		log.Fatalf("error: %s\n", err)
	}
}

func deleteHook(c *cli.Context) {
	if err := loadHooks(c); err != nil {
		log.Fatalf("error: %s\n", err)
	}

	terminateOnEmptyHooksFile()

	hooksToBeDeleted, err := findHooksByID(c)

	if err != nil {
		log.Fatalf("error: %s\n", err)
	}

	if len(hooksToBeDeleted) > 1 {
		if !c.IsSet("idx") && (!c.IsSet("all") || !c.Bool("all")) {
			log.Fatalf("there are %d hook(s) matching the given id\nuse --idx index to specify the one you want to delete\nor use --all to delete all of them\n", len(hooksToBeDeleted))
		} else {
			if !c.Bool("all") && c.IsSet("idx") {
				h, err := findOneHookByID(c)

				if err != nil {
					log.Fatalf("error: %s\n", err)
				}

				hooksToBeDeleted = append(make([]*hook.Hook, 0), h)
			}
		}
	}

	deleteHooks(hooksToBeDeleted)

	saveHooks()
}

func setHookProperties(h *hook.Hook, propertyValuePairs []string) error {
	for _, propertyValuePair := range propertyValuePairs {
		splitResult := strings.SplitN(propertyValuePair, "=", 2)

		if len(splitResult) != 2 {
			log.Fatalln("error: --set must follow property=newvalue format")
		}

		property, value := strings.ToLower(splitResult[0]), splitResult[1]

		switch {
		case property == "id":
			log.Printf(" + setting id to %s\n", value)
			h.ID = value

		case property == "execute-command":
			fallthrough
		case property == "cmd":
			log.Printf(" + setting execute-command to %s\n", value)
			h.ExecuteCommand = value

		case property == "trigger-rule":
			fallthrough
		case property == "rule":
			log.Printf(" + setting trigger-rule to %s\n", value)

			p := parser.New(value)

			if error := p.Parse(); error != nil {
				return error
			}

			h.TriggerRule = p.GeneratedRule
		case property == "command-working-directory":
			fallthrough
		case property == "cwd":
			log.Printf(" + setting command-working-directory to %s\n", value)
			h.CommandWorkingDirectory = value

		case property == "response-message":
			fallthrough
		case property == "message":
			log.Printf(" + setting response-message to %s\n", value)
			h.ResponseMessage = value

		/*case property == "pass-environment-to-command":
			fallthrough
		case property == "env":
			log.Println(" - removing pass-environment-to-command")
			h.PassEnvironmentToCommand = nil

		case property == "pass-arguments-to-command":
			fallthrough
		case property == "args":
			log.Println(" - removing pass-arguments-to-command")
			h.PassArgumentsToCommand = nil

		case property == "parse-parameters-as-json":
			fallthrough
		case property == "json-params":
			log.Println(" - removing parse-parameters-as-json")
			h.JSONStringParameters = nil*/

		case property == "include-command-output-in-response":
			fallthrough
		case property == "include-response":
			value = strings.ToLower(value)

			if value == "true" {
				log.Println(" + setting include-command-output-in-response to true")
				h.CaptureCommandOutput = true
			} else {
				if value == "false" {
					log.Println(" + setting include-command-output-in-response to false")
					h.CaptureCommandOutput = false
				} else {
					return fmt.Errorf("invalid value %s, expected true or false", value)
				}
			}

		default:
			return fmt.Errorf("invalid property name %s", property)
		}
	}

	return nil
}

func unsetHookProperties(h *hook.Hook, properties []string) error {
	for _, property := range properties {
		property = strings.ToLower(property)
		switch {
		case property == "id":
			log.Fatalln("error: property id is required")

		case property == "execute-command":
			fallthrough
		case property == "cmd":
			log.Println(" - removing execute-command")
			h.ExecuteCommand = ""

		case property == "trigger-rule":
			fallthrough
		case property == "rule":
			log.Println(" - removing trigger-rule")
			h.TriggerRule = nil

		case property == "command-working-directory":
			fallthrough
		case property == "cwd":
			log.Println(" - removing command-working-directory")
			h.CommandWorkingDirectory = ""

		case property == "response-message":
			fallthrough
		case property == "message":
			log.Println(" - removing response-message")
			h.ResponseMessage = ""

		case property == "pass-environment-to-command":
			fallthrough
		case property == "env":
			log.Println(" - removing pass-environment-to-command")
			h.PassEnvironmentToCommand = nil

		case property == "pass-arguments-to-command":
			fallthrough
		case property == "args":
			log.Println(" - removing pass-arguments-to-command")
			h.PassArgumentsToCommand = nil

		case property == "parse-parameters-as-json":
			fallthrough
		case property == "json-params":
			log.Println(" - removing parse-parameters-as-json")
			h.JSONStringParameters = nil

		case property == "include-command-output-in-response":
			fallthrough
		case property == "include-response":
			log.Println(" - removing include-command-output-in-response")
			h.CaptureCommandOutput = false

		default:
			return fmt.Errorf("invalid property name %s", property)
		}
	}

	return nil
}

func findHooksByID(c *cli.Context) ([]*hook.Hook, error) {
	if len(c.Args()) == 0 {
		return nil, fmt.Errorf("you must specify a valid hook id")
	}

	hooksSlice, ok := hooksMap[c.Args()[0]]

	if !ok {
		return nil, fmt.Errorf("could not find any hooks matching the given id")
	}

	return hooksSlice, nil
}

func findOneHookByID(c *cli.Context) (*hook.Hook, error) {
	hooksSlice, err := findHooksByID(c)

	if err != nil {
		return nil, err
	}

	if !c.IsSet("idx") && len(hooksSlice) > 1 {
		return nil, fmt.Errorf("there are %d hook(s) matching the given id\nuse --idx index to specify the one you want to modify", len(hooksSlice))
	}

	idx := c.Int("idx")

	if idx >= len(hooksSlice) || idx < 0 {
		return nil, fmt.Errorf("given local hook index is out of bounds")
	}

	return hooksSlice[idx], nil
}

func editHook(c *cli.Context) {
	if err := loadHooks(c); err != nil {
		log.Fatalf("error: %s\n", err)
	}

	terminateOnEmptyHooksFile()

	h, err := findOneHookByID(c)

	if err != nil {
		log.Fatalf("error: %s\n", err)
	}

	if err := unsetHookProperties(h, c.StringSlice("unset")); err != nil {
		log.Fatalf("error: cannot remove property: %s\n", err)
	}

	if err := setHookProperties(h, c.StringSlice("set")); err != nil {
		log.Fatalf("error: cannot set property: %s\n", err)
	}

	if err := saveHooks(); err != nil {
		log.Fatalf("error: %s\n", err)
	}
}

func init() {
	log.SetFlags(0)
}

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
			Aliases: []string{"ls", "show", "s"},
			Usage:   "lists hooks from the hooks file, or the hook(s) matching the given id",
			Action:  listHooks,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "expanded, e",
					Usage: "print expanded version when listing all hooks",
				},
				cli.BoolFlag{
					Name:  "compact, c",
					Usage: "print compact version of hook(s) matching the given id",
				},
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
			Aliases: []string{"e", "modify", "mod"},
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
					Name:  "unset, u",
					Usage: "property name",
				},
			},
		},
		{
			Name:    "add",
			Aliases: []string{"a", "new", "n", "create", "c"},
			Usage:   "creates a hook with the given id according to specified flags",
			Action:  addHook,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "set, s",
					Usage: "property=value",
				},
			},
		},
		{
			Name:    "delete",
			Aliases: []string{"del", "remove", "rm"},
			Usage:   "removes the hook matching the given id",
			Action:  deleteHook,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "idx, i",
					Value: 0,
					Usage: "local hook index (used for differentiating multiple hooks with the same id)",
				},
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "remove all hooks matching the given id",
				},
			},
		},
	}

	app.Run(os.Args)
}
