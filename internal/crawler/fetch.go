package crawler

import (
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
)

const (
	OK       = iota
	Redirect = iota
	NoData   = iota
)

type fetcher struct {
	client http.Client
}

func NewFetcher() *fetcher {
	var client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	return &fetcher{*client}
	//TODO Add request timeout
}

func (p *Page) Status() int {
	switch (*p).StatusCode / 100 {
	case 2:
		return OK
	case 3:
		return Redirect
	}
	return NoData
}

func (fetcher *fetcher) Fetch(task *Task) {
	resp, err := fetcher.client.Get(task.Page.URL.String())
	if err != nil {
		task.Error = err
		return
	} else {
		defer resp.Body.Close()
		task.Page.StatusCode = resp.StatusCode
		if task.ParseURLs == false {
			return
		}
		switch task.Page.Status() {
		case OK:
			depth := task.Page.Depth + 1
			for _, l := range getLinks(resp.Body) {
				u, err := url.Parse(l)
				if err != nil {
					log.Fatal(err)
				}
				absURL := task.Page.URL.ResolveReference(u)
				p := Page{absURL, depth, task.Page.URL.String(), 0}
				task.FoundPages = append(task.FoundPages, p)
			}
		case Redirect:
			redirectDestination, err := url.Parse(resp.Header.Get("Location"))
			if err != nil {
				task.Error = err
				return
			}
			p := Page{redirectDestination, task.Page.Depth, "Redirected from " + task.Page.URL.String(), 0}
			task.FoundPages = append(task.FoundPages, p)
		}
	}
}

func getLinks(body io.Reader) []string {
	var links []string
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return links
		case html.StartTagToken, html.EndTagToken:
			token := z.Token()
			if "a" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
	}
}