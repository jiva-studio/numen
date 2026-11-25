// Command seedidentity makes the test identities of the acceptance scenarios, or checks that the saved ones are
// what the code loads now. Without -check it writes the files into the folder of -out, unless the files there load
// already: the keys then stay as they are, and -force makes new ones.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/jiva-studio/numen/tools/syncserver/identityseed"
)

func main() {
	out := flag.String("out", "", "folder of the test identities")
	check := flag.Bool("check", false, "only check the saved files")
	force := flag.Bool("force", false, "write the files again although the saved ones load")
	flag.Parse()
	if *out == "" {
		log.Fatal("-out names the folder of the test identities")
	}
	if *check {
		if err := identityseed.Check(*out); err != nil {
			log.Fatal(err)
		}
		return
	}
	if !*force && identityseed.Check(*out) == nil {
		fmt.Println("the test identities are up to date:", *out)
		return
	}
	seed, err := identityseed.Generate()
	if err != nil {
		log.Fatal(err)
	}
	if err := identityseed.Write(*out, seed); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", *out)
}
