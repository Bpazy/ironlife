package cookiejar

import (
	"github.com/go-resty/resty/v2"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type ArcheryJar struct {
	*cookiejar.Jar
	client *resty.Client
}

func NewCookieJar(client *resty.Client) (*ArcheryJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &ArcheryJar{jar, client}, nil
}

func (j *ArcheryJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.Name == "csrftoken" {
			j.client.SetHeader("X-CSRFToken", cookie.Value)
		}
	}
	j.Jar.SetCookies(u, cookies)
}
