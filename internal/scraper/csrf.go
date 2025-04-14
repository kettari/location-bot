package scraper

import (
	"errors"
	"regexp"
)

type Csrf struct {
	page   *Page
	Token  string
	Cookie string
}

func NewCsrf(page *Page) *Csrf {
	return &Csrf{
		page: page,
	}
}

func (c *Csrf) ExtractCsrfToken() error {
	r := regexp.MustCompile(`<meta name="csrf-token" content="([\w\-=]+)">`)
	matches := r.FindAllStringSubmatch(c.page.Html, -1)
	if len(matches) > 0 && len(matches[0]) > 1 {
		c.Token = matches[0][1]
		return nil
	}
	return errors.New("csrf token 'csrf-token' not found in the HTML page")
}

func (c *Csrf) ExtractCsrfCookie() error {
	for _, cookie := range c.page.Cookies {
		if cookie.Name == "_csrf" {
			c.Cookie = cookie.Value
			return nil
		}
	}
	return errors.New("csrf cookie '_csrf' not found in the cookies")
}
