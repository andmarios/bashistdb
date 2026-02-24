/*
Package timestamp adds timestamps to an untimestamped
bash history. You may choose the starting date as X months
before now. Then the function will add timestamps at equal
intervals for every line in history.
*/
package timestamp

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Convert takes the content of a bash history file and number of months
// to span the non-timestamped lines accross.
// It processes the history line by line, adding timestamps where needed.
// Thus it is *safe* to run many times.
func Convert(history []byte, months int) []byte {
	hasTimestamp := regexp.MustCompile("^#")

	// Split input to lines
	lines := strings.Split(string(history), "\n")

	// Calculate time now, start time and num of lines equal spaces in between
	now := time.Now()
	since := now.AddDate(0, -1*months, 0)
	space := now.Sub(since) / time.Duration(len(lines))

	var out bytes.Buffer
	for i := 0; i < len(lines); i++ {
		if hasTimestamp.MatchString(lines[i]) { // Timestamped, just copy
			if i+1 < len(lines) {
				out.WriteString(fmt.Sprintf("%s\n%s\n", lines[i], lines[i+1]))
				i++
			} else {
				out.WriteString(fmt.Sprintf("%s\n", lines[i]))
			}
			continue
		}
		if lines[i] != "" { // Line not timestamped and not empty?
			out.WriteString(fmt.Sprintf("#%d\n", since.Unix()+int64(i)*int64(space.Seconds())))
			out.WriteString(fmt.Sprintf("%s\n", lines[i]))
		}
	}
	return out.Bytes()
}
