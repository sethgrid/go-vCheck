package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

var srcDir string

type VersionInfo struct {
	Name, LocalVersion, RemoteVersion string
}

type GitHubAPIResponse struct {
	// technically, we only need Content
	Name     string `json:"name"`
	Path     string `json:"path"`
	Sha      string `json:"sha"`
	Size     int    `json:"size"`
	Url      string `json:"url"`
	HtmlUrl  string `json:"html_url"`
	GitUrl   string `json:"git_url"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Links    string `json:"_links"`
	Git      string `json:"git"`
	Html     string `json:"html"`
}

func init() {
	flag.StringVar(&srcDir, "src", "src/",
		"Checks all SendGrid repos in src/ to compare "+
			"local verisions with GitHub. User can set -src= "+
			"or just pass the path as the first command line argument.")

	// flag.Args is not getting back commandline args.
	// The user can pass -src=/path/to/src or just pass /path/to/src/
	if len(os.Args) >= 2 && !strings.Contains(os.Args[1], "-src") {
		srcDir = os.Args[1]
	}

}

func main() {
	flag.Parse()
	srcDir = addTrailingSlash(srcDir)
	oauth_token := ""

	env := os.Environ()
	for _, e := range env {
		line := strings.Split(e, "=")
		if line[0] == "SG_GITHUB_TOKEN" {
			oauth_token = line[1]
		}
	}
	if oauth_token == "" {
		friendlyExit("you must set the environment variable SG_GITHUB_TOKEN")
	}

	fmt.Println("Checking all Sendgrid packages round in:", srcDir)
	repos := getSendGridRepos(srcDir)
	results := make([]*VersionInfo, 0)

	wg := &sync.WaitGroup{}
	for _, repo := range repos {
		// if you don't have the os.env for the sendgrid github oauth token,
		// you can replace the url below with your own fork and use your
		// own token.
		wg.Add(2)
		info := &VersionInfo{Name: repo}

		go func(repo string) {
			go func() {
				info.LocalVersion = GetLocalVersion(srcDir + "github.com/sendgrid/" + repo + "/version.go")
				wg.Done()
			}()
			go func() {
				info.RemoteVersion = GetRemoteVersion("https://api.github.com/repos/sendgrid/"+repo+"/contents/version.go?ref=master", oauth_token)
				wg.Done()
			}()
		}(repo)

		results = append(results, info)
	}
	wg.Wait()

	for _, r := range results {
		fmt.Printf("%s\n  Local:  %s\n  GitHub: %s\n",
			r.Name, r.LocalVersion, r.RemoteVersion)
	}
}

// read the specified file and find the version
func GetLocalVersion(filePath string) string {
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		friendlyExit(err.Error())
	}

	return MatchVersion(contents)
}

// using the OAuth token, look at GitHub repo and find the version
func GetRemoteVersion(httpPath, token string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", httpPath, nil)
	if err != nil {
		friendlyExit(err.Error())
	}
	req.Header.Add("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		friendlyExit(err.Error())
	}

	if resp.StatusCode == 404 {
		friendlyExit("GitHub 404. Did you use the correct token?")
	}

	content, err := ioutil.ReadAll(resp.Body)
	jsonResp := &GitHubAPIResponse{}
	json.Unmarshal(content, jsonResp)

	decoded, err := base64.StdEncoding.DecodeString(jsonResp.Content)
	if err != nil {
		friendlyExit(err.Error())
	}

	return MatchVersion(decoded)
}

// regex match to find semantic version
func MatchVersion(source []byte) string {
	expr := `VERSION.+(\d.\d.\d)`
	r, err := regexp.Compile(expr)
	if err != nil {
		friendlyExit(err.Error())
	}
	matches := r.FindSubmatch(source)
	if len(matches) >= 2 {
		return string(matches[1])
	}
	return "ERROR - no match found for " + expr + ". Did we change format?"
}

// allow our user to optionally include their own parenthesis.
func addTrailingSlash(path string) string {
	if string(path[len(path)-1]) == "/" {
		return path
	}
	return path + "/"
}

// Assuming everything is in ../github.com/sendgrid
func getSendGridRepos(srcDir string) []string {
	contents, err := ioutil.ReadDir(srcDir + "github.com/sendgrid")
	if err != nil {
		friendlyExit(err.Error())
	}

	repos := make([]string, 0)
	for _, item := range contents {
		if item.IsDir() {
			repos = append(repos, item.Name())
		}
	}
	return repos
}

// give some color and useful usage information before exiting.
func friendlyExit(errMsg string) {
	// set output to bold red
	fmt.Printf("\n\x1b[31;1m%s\x1b[0m\n", errMsg)
	usage()
	os.Exit(1)
}

func usage() {
	fmt.Println(`
Example Usage:
	$ export SG_GITHUB_TOKEN="[:token]"
	$ vCheck
	or
	$ SG_GITHUB_TOKEN="[:token]" vCheck

vCheck defaults to look in the current directory for src/. However,
you can explicitly set where vCheck should look. Examples:
	$ cd /path/to/project; vCheck
	$ vCheck /path/to/project/src
	$ vCheck -src=/path/to/project/src

Environment Variable
vCheck uses a Personal GitHub Access Token to make http requests to GitHub.
See http://developer.github.com/v3/auth/#basic-authentication for OAuth Tokens.

You can set the environment variable by exporting it or setting it when calling
vCheck as shown in the example usage.
`)
}
