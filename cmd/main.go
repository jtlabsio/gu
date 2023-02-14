package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/jessevdk/go-flags"
	"golang.org/x/net/html"
)

const (
	DefaultDownloadValue string = "unknown"
	GODownloadsURL       string = "https://go.dev/dl/"
)

var (
	cs                = ""
	isArchive         = regexp.MustCompile(`archive`)
	isDownloadRE      = regexp.MustCompile(`download`)
	isFeaturedRE      = regexp.MustCompile(`featured`)
	isPackageRE       = regexp.MustCompile(`\.pkg`)
	isUnstableRE      = regexp.MustCompile(`unstable`)
	latestRE          = regexp.MustCompile(`feature|latest|stable`)
	platformVersionRE = regexp.MustCompile(`\/dl\/go([a-z0-9\.]*)\.(darwin|freebsd|linux|src|windows){1}(\-([a-z0-9]*))?`)
	sectionRE         = regexp.MustCompile(`(archive|featured|stable|unstable)`)
)

type cmdOpts struct {
	Archived bool `short:"a" long:"archive" description:"Include archived Go versions" required:"false"`
	Args     struct {
		Version []string
	} `positional-args:"yes" required:"yes"`
	Featured bool `short:"f" long:"featured" description:"Install featured version" required:"false"`
	List     bool `short:"l" long:"ls" description:"Available Go versions" required:"false"`
}

func (co cmdOpts) ShowUsage() bool {
	return !co.List && len(co.Args.Version) == 0
}

type download struct {
	Arch     string
	Archive  bool
	Featured bool
	OS       string
	Unstable bool
	Url      string
	Version  string
}

func (d download) Installable() bool {
	// source code is not installable
	if d.OS == "src" {
		return false
	}

	// match architecture
	if d.Arch != runtime.GOARCH {
		return false
	}

	// match OS
	if d.OS != runtime.GOOS {
		return false
	}

	return true
}

func (d download) Platform() string {
	return strings.Join([]string{d.OS, d.Arch}, " ")
}

func extractDownloadLinks(n *html.Node) []download {
	lnks := []download{}

	// is this an element node?
	if n.Type == html.ElementNode {
		// check for section starts...
		if n.Data == "h2" || n.Data == "div" {
			// read the link's attributes
			for _, a := range n.Attr {
				// check for featured, unstable, or stable section start
				if a.Key == "id" && sectionRE.MatchString(a.Val) {
					cs = a.Val
				}
			}
		}

		// look for links
		if n.Data == "a" {
			d := false
			l := ""

			// read the link's attributes
			for _, a := range n.Attr {
				// check for a download
				if a.Key == "class" {
					d = isDownloadRE.MatchString(a.Val)
				}

				// grab the href
				if a.Key == "href" {
					l = a.Val
				}
			}

			if d {
				dl := fromLink(l)
				dl.Archive = isArchive.MatchString(cs)
				dl.Featured = isFeaturedRE.MatchString(cs)
				dl.Unstable = isUnstableRE.MatchString(cs)

				return []download{dl}
			}
		}
	}

	// depth-first traversal of children
	for cn := n.FirstChild; cn != nil; cn = cn.NextSibling {
		tlnks := extractDownloadLinks(cn)

		// de-dupe while processing (featured links are duplicated)
		if len(tlnks) > 0 {
			dup := false
			for _, lnk := range lnks {
				if lnk.Url == tlnks[0].Url {
					dup = true
					break
				}
			}

			if !dup {
				lnks = append(lnks, tlnks...)
			}
		}
	}

	return lnks
}

func fromLink(l string) download {
	url, _ := url.JoinPath(GODownloadsURL, path.Base(l))

	// mac os patch: replace .pkg w/ .tar.gz
	if isPackageRE.MatchString(url) {
		url = isPackageRE.ReplaceAllString(url, ".tar.gz")
	}

	d := download{
		Arch: DefaultDownloadValue,
		OS:   DefaultDownloadValue,
		Url:  url,
	}

	pp := platformVersionRE.FindAllStringSubmatch(l, -1)
	if len(pp) > 0 {
		// set Version
		d.Version = pp[0][1]

		// set OS
		d.OS = pp[0][2]

		// set arch
		if len(pp[0]) > 3 {
			d.Arch = pp[0][4]
		}
	}

	return d
}

func installVersion(dl download) {
	// download .tar.gz from the link
	res, err := http.Get(dl.Url)
	if err != nil {
		log.Panic(err)
	}
	defer res.Body.Close()

	// remove any existing installed versions of go (GOROOT)
	if err := os.RemoveAll(runtime.GOROOT()); err != nil {
		log.Panicf(
			"remove previously installed version (%s) failed: %s",
			runtime.GOROOT(),
			err.Error())
	}

	// identify path to install go
	df := path.Dir(runtime.GOROOT())

	// open gzip reader
	us, err := gzip.NewReader(res.Body)
	if err != nil {
		log.Panicf("open gzip failed: %s", err.Error())
	}

	// open tar reader from the gzip and begin processing download
	tr := tar.NewReader(us)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Panicf("extract tar from gzip: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// create temp folder
			if err := os.Mkdir(path.Join(df, header.Name), 0755); err != nil {
				log.Panicf("extract tar from gzip: Mkdir() failed: %s", err.Error())
			}

		case tar.TypeReg:
			// create temp file
			of, err := os.OpenFile(path.Join(df, header.Name), os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				log.Panicf("extract tar from gzip: Create() failed: %s", err.Error())
			}

			// copy contents to temp file
			if _, err := io.Copy(of, tr); err != nil {
				log.Panicf("extract tar from gzip: Copy() failed: %s", err.Error())
			}
			of.Close()

		default:
			log.Panicf(
				"extract tar from gzip: unknown type %b in %s",
				header.Typeflag,
				header.Name)
		}
	}

	fmt.Printf("installed version %s locally to %s\n", dl.Version, df)
}

func showAvailable(lnks []download, ia bool) {
	for _, l := range lnks {
		// only print installable non-archived options
		if !l.Installable() || (l.Archive && !ia) {
			continue
		}

		// highlight featured...
		fs := ""
		if l.Featured {
			fs = "(featured)"
		}

		if l.Unstable {
			fs = "(unstable)"
		}

		// spacing...
		sp := "\t\t"
		if len(l.Version) >= 8 {
			sp = "\t"
		}

		// list installable versions
		fmt.Printf("%s%s[%s] %s\n", l.Version, sp, l.Platform(), fs)
	}

	fmt.Println()
}

func showUsage() {
	// get executable name
	ex, _ := os.Executable()
	ex = filepath.Base(ex)

	// format the usage
	fmt.Printf("Usage:\n\t%s [OPTIONS] Version...\n", ex)
	fmt.Println("Upgrades currently installed version of Go")
	fmt.Println()
	fmt.Println("Application Options:")
	fmt.Println("  -a, --archived  Include archived Go versions")
	fmt.Println("  -f, --featured  Install featured version")
	fmt.Println("  -l, --ls        Available Go versions")
	fmt.Println()
	fmt.Println("Help Options:")
	fmt.Println("  -h, --help      Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  Install Go version 1.19.4:\n    %s 1.19.4\n", ex)
	fmt.Println()
	fmt.Printf("  Show archived Go download options:\n    %s -la\n", ex)
	fmt.Println()
}

func main() {
	var (
		opts cmdOpts
		prsr = flags.NewParser(&opts, flags.Default)
	)

	// parse the command line arguments
	if _, err := prsr.Parse(); err != nil {
		os.Exit(1)
	}

	// show usage
	if opts.ShowUsage() {
		showUsage()
		os.Exit(1)
	}

	// retrieve the go download links
	res, err := http.Get(GODownloadsURL)
	if err != nil {
		log.Panic(err)
	}
	defer res.Body.Close()

	// ensure we have a successful response
	if res.StatusCode != 200 {
		log.Panic(
			fmt.Errorf(
				"unexpected status received while requesting downloads from %s: %d",
				GODownloadsURL,
				res.StatusCode))
	}

	// creater parser for the markup that was returned
	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Panic(err)
	}

	// extract the download links using the parser
	lnks := extractDownloadLinks(doc)

	// should we show available installs?
	if opts.List {
		showAvailable(lnks, opts.Archived)
		os.Exit(0)
	}

	var dl download

	// validate that version is available
	for _, d := range lnks {
		// check for version match
		if d.Version == opts.Args.Version[0] && d.Installable() {
			dl = d
			break
		}

		// check for featured / stable
		if latestRE.MatchString(opts.Args.Version[0]) && d.Featured && d.Installable() {
			dl = d
		}

		// check for unstable
		if isUnstableRE.MatchString(opts.Args.Version[0]) && d.Unstable && d.Installable() {
			dl = d
		}
	}

	// now installable version found with provided value
	if dl == (download{}) {
		fmt.Printf("requested version (%s) not found\n", opts.Args.Version[0])
		os.Exit(1)
	}

	fmt.Printf("installing version %s (%s)\n", dl.Version, dl.Url)
	installVersion(dl)
}
