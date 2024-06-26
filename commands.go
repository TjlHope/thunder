/*
 * Thunder, BoltDB's interactive shell
 *     Copyright (c) 2017, Christian Muehlhaeuser <muesli@gmail.com>
 *     Copyright (c) 2023, Tom Hope <tjlhope@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/muesli/ishell"
	jsonpointer "github.com/dustin/go-jsonpointer"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

func travel(cwd Bucket, path string) (Bucket, error) {
	var err error
	parts := strings.Split(path, "/")
	for i := 0; err == nil && cwd != nil && i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}

		part := parts[i]
		if part == ".." {
			if cwd.Prev() != nil {
				cwd = cwd.Prev()
			}
		} else if part != "." {
			cwd, err = cwd.Cd(part)
		}
	}

	return cwd, err
}

func parseKeyPath(cwd Bucket, path string) (Bucket, string, error) {
	slashIndex := strings.LastIndex(path, "/")
	var key string
	var err error
	if slashIndex < 0 {
		key = path
	} else {
		key = path[slashIndex+1:]
		cwd, err = travel(cwd, path[:slashIndex])
	}
	return cwd, key, err
}

func lsCmd(c *ishell.Context) {
	target := cwd
	if len(c.Args) > 0 {
		var err error
		target, err = travel(target, c.Args[0])
		if err != nil {
			c.Err(err)
			return
		}
	}

	contents := target.List()
	entries := printableList(contents)
	for _, entry := range entries {
		c.Println(entry)
	}

	if c.Get("mode") == Interactive {
		footnote := ""
		omitted := len(contents) - len(entries)
		if omitted > 0 {
			footnote = fmt.Sprintf(" (%d omitted in this list)", omitted)
		}
		c.Printf("%d keys in bucket%s\n", len(contents), footnote)
	}
}

func getCmd(c *ishell.Context) {
	switch len(c.Args) {
	case 0:
		c.Err(errors.New("get: missing key name"))
		return
	case 1: // <key>
		break
	case 3:	// <key> --json <path:json-pointer>
		if c.Args[1] == "--json" {
			break
		} // fall-through
	default:
		c.Err(errors.New("get: too many arguments"))
		return
	}

	var data []byte
	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	data, err = target.Get(key)
	if err != nil {
		c.Err(err)
		return
	}

	if len(c.Args) == 3 && c.Args[1] == "--json" {
		data, err = jsonpointer.Find(data, c.Args[2])
		if err != nil {
			c.Err(err)
			return
		}
	}

	c.Println(string(data))
}

func patchJsonValue(bucket Bucket, key string, patch []byte) (error) {
	doc, err := bucket.Get(key)
	if err != nil { return err }
	patcher, err := jsonpatch.DecodePatch(patch)
	if err != nil { return err }
	modified, err := patcher.Apply(doc)
	if err != nil { return err }
	return bucket.Put(key, string(modified))
}

func putCmd(c *ishell.Context) {
	switch len(c.Args) {
	case 0:
		c.Err(errors.New("put: missing key name and value"))
		return
	case 1:
		c.Err(errors.New("put: missing value"))
		return
	case 2: // <key> <value>
		break
	case 3: // <key> --json-patch <patch>
		if c.Args[1] != "--json-patch" {
			c.Err(errors.New("put: expected: <key> --json-patch <patch>"))
			return
		}
		break
	case 4: // <key> --json <path:json-pointer> <value>
		if c.Args[1] != "--json" {
			c.Err(errors.New("put: expected: <key> --json-set <path> <value>"))
			return
		}
		break
	default:
		c.Err(errors.New("put: too many arguments"))
		return
	}

	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	switch c.Args[1] {
	case "--json":
		patchJSON := []byte(fmt.Sprintf(
			`[{"op": "add", "path": "%s", "value": %s}]`,
			c.Args[2], c.Args[3]))
		err = patchJsonValue(target, key, patchJSON)
		break
	case "--json-patch":
		err = patchJsonValue(target, key, []byte(c.Args[2]))
		break
	default:
		err = target.Put(key, c.Args[1])
		break
	}

	c.Err(err)
}

func cdCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		/* go to root */
		for cwd.Prev() != nil {
			cwd = cwd.Prev()
		}
	} else {
		b, err := travel(cwd, c.Args[0])
		if err != nil {
			c.Err(err)
			return
		}
		cwd = b
	}
	setPrompt(c)
}

func setPrompt(c *ishell.Context) {
	if c.Get("mode") == Interactive {
		shell.SetPrompt(fmt.Sprintf(promptFmt, fname, cwd.String()))
	} else {
		shell.SetPrompt("")
	}
}

func mkdirCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Err(errors.New("mkdir: missing bucket name"))
		return
	}

	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	c.Err(target.Mkdir(key))
}

func rmCmd(c *ishell.Context) {
	if len(c.Args) < 1 {
		c.Err(errors.New("rm: missing bucket or key name"))
		return
	}

	target, key, err := parseKeyPath(cwd, c.Args[0])
	if err != nil {
		c.Err(err)
		return
	}

	c.Err(target.Rm(key))
}

func modeCmd(c *ishell.Context) {
	if len(c.Args) == 1 {
		arg := c.Args[0]
		if IsMode(arg) {
			c.Set("mode", arg)
			setPrompt(c)
			return
		}
	}
	c.Err(errors.New(fmt.Sprintf("mode: single argument needed: %s", Modes())))
}
