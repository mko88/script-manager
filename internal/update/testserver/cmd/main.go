// Command update-test-server serves the binaries in a directory as a fake
// latest release, for trying the in-app updater by hand. Developer tool; not
// shipped.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"script-manager/internal/update"
	"script-manager/internal/update/testserver"
)

func main() {
	dir := flag.String("dir", "", "directory holding the new script-manager, script-manager-gui and sm-config-edit binaries")
	tag := flag.String("tag", "v9.9.9", "release tag to advertise")
	addr := flag.String("addr", "127.0.0.1:8765", "listen address")
	corrupt := flag.Bool("bad-checksum", false, "publish a wrong checksum for every archive")
	noSums := flag.Bool("no-checksums", false, "publish no checksums.txt")
	pre := flag.Bool("prerelease", false, "mark the release as a prerelease")
	flag.Parse()

	if *dir == "" {
		flag.Usage()
		os.Exit(2)
	}
	rel, err := testserver.FromDir(*dir, *tag)
	if err != nil {
		log.Fatal(err)
	}
	rel.Prerelease = *pre
	switch {
	case *noSums:
		delete(rel.Assets, update.ChecksumsAsset)
	case *corrupt:
		var names []string
		for n := range rel.Assets {
			if n != update.ChecksumsAsset {
				names = append(names, n)
			}
		}
		bogus := map[string][]byte{}
		for _, n := range names {
			bogus[n] = []byte("not the archive")
		}
		rel.Assets[update.ChecksumsAsset] = testserver.Checksums(bogus, names...)
	}

	for name := range rel.Assets {
		fmt.Println("asset:", name)
	}
	fmt.Printf("serving %s on http://%s\n", *tag, *addr)
	fmt.Printf("set %s=http://%s before starting the app\n", update.EnvBaseURL, *addr)
	log.Fatal(http.ListenAndServe(*addr, testserver.Handler(rel, func(r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
	})))
}
