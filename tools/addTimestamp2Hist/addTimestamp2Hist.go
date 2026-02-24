/*
Command addTimestamp2Hist adds timestamps to an untimestamped
bash history file. You may choose the starting date as X months
before now. Then the utility will add timestamps at equal
intervals for every line in history.

	$ addTimestamp2Hist ~/.bash_history
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andmarios/bashistdb/tools/addTimestamp2Hist/timestamp"
)

var home = os.Getenv("HOME")

var (
	historyFile = flag.String("f", home+"/.bash_history", "file to process")
	duration    = flag.Int("since", 3, "span timestamps in equal spaces between N months and now")
	write       = flag.Bool("write", false,
		"if set will overwrite the bash history file with the timestamped version,"+
			"        default behaviour is to print to stdout")
)

func main() {
	flag.Parse()

	historyIn, err := os.ReadFile(*historyFile)
	if err != nil {
		log.Fatalln(err)
	}
	historyOut := timestamp.Convert(historyIn, *duration)

	if !*write {
		fmt.Println(string(historyOut))
	} else {
		err = os.WriteFile(*historyFile, historyOut, 0600)

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("History file converted.")
		}
	}
}
