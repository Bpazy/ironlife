package ironlife

import "strings"

func CompleteProtocol(url string) string {
	if strings.HasPrefix(url, "http://") &&
		strings.HasPrefix(url, "https://") {
		return url
	}
	return "http://" + url
}
