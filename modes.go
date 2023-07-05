/*
 * Thunder, BoltDB's interactive shell
 *     Copyright (c) 2023, Tom Hope <tjlhope@gmail.com>
 *
 *   For license see LICENSE
 */

package main

type Mode string

const (
	Batch       Mode = "batch"
	Interactive      = "interactive"
)

func Modes() []Mode {
	return []Mode{Batch, Interactive}
}

func IsMode(val string) bool {
	switch Mode(val) {
	case Batch, Interactive:
		return true
	}
	return false
}
