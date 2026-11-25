// Command seedidentity makes the test identities of the acceptance scenarios, or checks that the saved ones are
// what the code makes now. Without -check it writes the files into the folder of -out.
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
	seed, err := identityseed.Generate()
	if err != nil {
		log.Fatal(err)
	}
	if err := identityseed.Write(*out, seed); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", *out)
}
